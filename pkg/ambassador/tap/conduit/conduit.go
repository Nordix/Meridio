/*
Copyright (c) 2021-2023 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conduit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/conduit"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/retry"
	grpcCodes "google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Conduit implements types.Conduit (/pkg/ambassador/tap/types)
// Responsible for requesting/closing the NSM Connection to the conduit,
// managing the streams and configuring the VIPs.
type Conduit struct {
	// Should be a unique name
	TargetName string
	// Namespace of the trench
	Namespace string
	Conduit   *ambassadorAPI.Conduit
	// Node name the pod is running on
	NodeName                string
	NetworkServiceClient    networkservice.NetworkServiceClient
	MonitorConnectionClient networkservice.MonitorConnectionClient
	Configuration           Configuration
	StreamManager           StreamManager
	NetUtils                networking.Utils
	StreamFactory           StreamFactory
	connection              *networkservice.Connection
	mu                      sync.Mutex
	// IPs assigned to the NSM connection on this side, fetched to get announced as Target IPs to NSP
	localIPs []string
	// VIP addresses received from NSP and already configured on the NSM connection
	vips                    []string
	logger                  logr.Logger
	monitorConnectionCancel context.CancelFunc
	nspEntryTimeout         time.Duration
}

// New is the constructor of Conduit.
// The constructor will create a new stream factory and a VIP configuration watcher
func New(conduit *ambassadorAPI.Conduit,
	targetName string,
	namespace string,
	nodeName string,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	targetRegistryClient nspAPI.TargetRegistryClient,
	networkServiceClient networkservice.NetworkServiceClient,
	monitorConnectionClient networkservice.MonitorConnectionClient,
	streamRegistry types.Registry,
	netUtils networking.Utils,
	nspEntryTimeout time.Duration) (*Conduit, error) {
	logger := log.Logger.WithValues("class", "Conduit", "conduit", conduit, "namespace", namespace, "node", nodeName)
	logger.Info("Create conduit")
	c := &Conduit{
		TargetName:              targetName,
		Namespace:               namespace,
		Conduit:                 conduit,
		NodeName:                nodeName,
		NetworkServiceClient:    networkServiceClient,
		MonitorConnectionClient: monitorConnectionClient,
		NetUtils:                netUtils,
		connection:              nil,
		localIPs:                []string{},
		vips:                    []string{},
		logger:                  logger,
		nspEntryTimeout:         nspEntryTimeout,
	}
	c.StreamFactory = stream.NewFactory(targetRegistryClient, c)
	c.StreamManager = NewStreamManager(configurationManagerClient, targetRegistryClient, streamRegistry, c.StreamFactory, PendingTime, nspEntryTimeout)
	c.Configuration = newConfigurationImpl(c.StreamManager.SetStreams, c.Conduit.ToNSP(), configurationManagerClient)
	return c, nil
}

// Connect requests the connection to NSM and, if success, will open all streams added
// and watch the VIPs.
// Will also try to query NSM if a connection with the same ID already exists. If it does,
// try to re-use that connection to avoid interference (e.g., when old connection's token
// lifetime expires).
//
// Rational behind using the same connection (segment) ID on the TAPA side:
// IPs assigned to the NSM connection might be restored even if the Proxy side segment ID changes
// (e.g., due to proxy kill, upgrade etc.). Thus, there might be no need to update localIPs,
// hence avoiding update of NSP and LBs about Target IP changes. It also avoids leaking the IPs
// of "old" TAPA->Proxy connections if the proxy POD has been replaced.
//
// If connection was not re-used with the help of connectionMonitor:
// - Token expiration of "old" connection would lead to a heal event, which could trigger
// reconnect (depends on both datapath monitoring state and results).
// - Reconnect re-creates the NSM interfaces. Likely resulting in new MAC addresses
// causing traffic disturbances even if "old" localIPs were kept (mostly due to
// the neighbor cache in LB).
// - Tear down of old connection would make the IPAM release the IPs shared by two connection.
// (Released IPs could be re-acquired shortly in case of reconnect. But without reconnect IPs
// could get re-assigned more likely to some other connection.)
func (c *Conduit) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isConnected() {
		return nil
	}

	var request *networkservice.NetworkServiceRequest
	var monitoredConnections map[string]*networkservice.Connection
	var nsName string = conduit.GetNetworkServiceNameWithProxy(c.Conduit.GetName(), c.Conduit.GetTrench().GetName(), c.Namespace)
	var id string = fmt.Sprintf("%s-%s-%d", c.TargetName, nsName, 0)
	c.logger.Info("Connect", "ID", id)

	// initial request
	request = &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             id,
			NetworkService: nsName,
			Labels: map[string]string{
				"nodeName": c.NodeName, // required to connect to the proxy on same node
			},
			Payload: payload.Ethernet,
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:        cls.LOCAL,
				Type:       kernelmech.MECHANISM,
				Parameters: map[string]string{},
			},
		},
	}

	if c.MonitorConnectionClient != nil {
		// check if NSM already tracks a connection with the same ID, if it does, re-use the connection
		stream, err := c.MonitorConnectionClient.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{
				{
					Id: id,
				},
			},
		})
		if err != nil {
			c.logger.Error(err, "Connect failed to create monitorConnectionClient")
			return fmt.Errorf("failed to create monitor connection client: %w", err)
		}

		event, err := stream.Recv()
		if err != nil {
			// probably running older NSM version, fallback to legacy behavior to request connection
			c.logger.Error(err, "error from monitorConnection stream")
		} else {
			monitoredConnections = event.Connections
		}
	}

	for _, conn := range monitoredConnections {
		path := conn.GetPath()
		// XXX: surprisingly, I still got connections that did not match the filter ID.
		if path != nil && path.Index == 1 && path.PathSegments[0].Id == id && conn.Mechanism.Type == request.MechanismPreferences[0].Type {
			c.logger.Info("Connect recovered connection", "connection", conn)
			// Keeping Policy Routes and VIPs if any as of now. Can mitigate impact of TAPA container crash. (I see no other benefit keeping them.
			// Note: Might also recover connection that belonged to a recently redeployed trench.
			// TODO: Maybe ignore Policy Routes and VIPs if last update time of connection had to be over 50% of our MaxTokenLifetime.

			// conn.Context.IpContext.Policies = []*networkservice.PolicyRoute{}
			// conn.Context.IpContext.SrcIpAddrs = getLocalIPs(
			// 	conn.GetContext().GetIpContext().GetSrcIpAddrs(),
			// 	c.vips, conn.GetContext().GetIpContext().GetExtraPrefixes(),
			// )

			// make sure to connect a local proxy
			if conn.Labels == nil {
				conn.Labels = request.Connection.Labels
			} else {
				conn.Labels["nodeName"] = c.NodeName
			}

			// update request based on connection
			request.Connection = conn
			request.Connection.Path.Index = 0
			request.Connection.Id = id
			if conn.Mechanism != nil && conn.Mechanism.GetParameters() != nil {
				if val, ok := conn.Mechanism.GetParameters()[common.InterfaceNameKey]; ok {
					// Recovered interface name could be taken by some other connection by now.
					// But we shall try to use it to avoid possible issues due to interface name updates.
					// (For example even with NSM 1.10 the Policy Based routing tables ended up empty if
					// the interface name was updated after TAPA crash.)
					// Remove the interface name from the connection to force our custom interfaceNameClient
					// to process the NSM request when sent. Also, pass the recovered interface name as the
					// preferred value via MechanismPreferences to interfaceNameClient.
					request.MechanismPreferences[0].Parameters[common.InterfaceNameKey] = val
					delete(conn.Mechanism.GetParameters(), common.InterfaceNameKey)
				}
			}
			break
		}
	}

	// Check if recovered connection indicates issue with control plane,
	// if so request reselect. Otherwise, the connection request might
	// fail if an old path segment (e.g. NSE) was replaced in the meantime
	// and recovery would take much longer.
	// (refer to https://github.com/networkservicemesh/cmd-nsc/pull/600)
	if request.GetConnection().State == networkservice.State_DOWN {
		c.logger.Info("Connect requesting reselect for recovered connection")
		request.GetConnection().Mechanism = nil
		request.GetConnection().NetworkServiceEndpointName = ""
		request.GetConnection().State = networkservice.State_RESELECT_REQUESTED
	}

	// request the connection
	originalRequest := request.Clone()
	connection, err := c.NetworkServiceClient.Request(ctx, request)
	if err != nil {
		err := fmt.Errorf("nsc connection request error: %w", err)
		return fmt.Errorf("%w; original request: %v", err, originalRequest)
	}
	c.connection = connection
	if len(c.getGateways()) > 0 {
		// Filter out any VIPs from local IPs by relying on gateway subnets,
		// in order to avoid announcing VIPs as Target IPs to NSP (if connection
		// was recovered using monitorConnectionClient).
		c.localIPs = getLocalIPs(c.connection.GetContext().GetIpContext().GetSrcIpAddrs(), c.vips, c.getGateways())
	} else {
		c.localIPs = c.connection.GetContext().GetIpContext().GetSrcIpAddrs()
	}
	c.logger.Info("Connected", "connection", connection, "localIPs", c.localIPs)

	c.Configuration.Watch()

	var ctxMonitorConnection context.Context
	ctxMonitorConnection, c.monitorConnectionCancel = context.WithCancel(context.Background())
	go c.monitorConnection(ctxMonitorConnection, connection)

	c.StreamManager.Run()
	return nil
}

// Disconnect closes the connection from NSM, closes all streams
// and stop the VIP watcher
func (c *Conduit) Disconnect(ctx context.Context) error {
	c.logger.Info("Disconnect")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.monitorConnectionCancel != nil {
		c.monitorConnectionCancel()
	}
	// Stops the configuration
	c.Configuration.Stop()
	var errFinal error
	// Stop the stream manager (close the streams)
	errFinal = c.StreamManager.Stop(ctx)
	// Close the NSM connection
	if c.isConnected() {
		_, err := c.NetworkServiceClient.Close(ctx, c.connection)
		if err != nil {
			errFinal = fmt.Errorf("%w; nsc connection close error: %w", errFinal, err) // todo
		}
	}
	c.connection = nil
	c.vips = []string{}
	return errFinal
}

// AddStream creates a stream based on its factory and will open it (in another goroutine)
func (c *Conduit) AddStream(ctx context.Context, strm *ambassadorAPI.Stream) error {
	c.logger.Info("Add stream", "stream", strm)
	if !c.Equals(strm.GetConduit()) {
		return errors.New("invalid stream for this conduit")
	}
	if err := c.StreamManager.AddStream(strm); err != nil {
		return fmt.Errorf("conduit stream manager failed to add stream: %w", err)
	}
	return nil
}

// RemoveStream closes and removes the stream (if existing), and removes it from the
// stream registry.
func (c *Conduit) RemoveStream(ctx context.Context, strm *ambassadorAPI.Stream) error {
	c.logger.Info("Remove stream", "stream", strm)
	if err := c.StreamManager.RemoveStream(ctx, strm); err != nil {
		return fmt.Errorf("conduit stream manager failed to remove stream: %w", err)
	}
	return nil
}

// GetStreams returns all streams previously added to this conduit
func (c *Conduit) GetStreams() []*ambassadorAPI.Stream {
	return c.StreamManager.GetStreams()
}

func (c *Conduit) GetConduit() *ambassadorAPI.Conduit {
	return c.Conduit
}

// GetIPs returns the local IPs for this conduit
func (c *Conduit) GetIPs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connection != nil {
		return c.localIPs
	}
	return []string{}
}

// SetVIPs checks the vips which has to be added or removed
func (c *Conduit) SetVIPs(ctx context.Context, vips []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isConnected() {
		return nil
	}
	// remove VIPs overlapping with internal subnets.
	vipsInGatewaySubnet := getIPsInGatewaySubnet(vips, c.getGateways())
	vipsInGatewaySubnetSet := sets.New(vipsInGatewaySubnet...)
	vipsSet := sets.New(formatPrefixes(vips)...)
	if vipsSet.Equal(sets.New(c.vips...)) {
		// same set of vips, skip NSM connection update
		// note: NSM heal related close and reconnect seems to keep IpContext intact (including VIP setup)
		return nil
	}
	c.logger.Info("SetVIPs", "VIPs", vips)
	vips = vipsSet.Difference(vipsInGatewaySubnetSet).UnsortedList()
	// prepare SrcIpAddrs (IPs allocated by the proxy + VIPs)
	c.connection.Context.IpContext.SrcIpAddrs = append(c.localIPs, vips...)
	// prepare the routes (nexthops = proxy bridge IPs)
	ipv4Nexthops := []*networkservice.Route{}
	ipv6Nexthops := []*networkservice.Route{}
	for _, nexthop := range c.getGateways() {
		gw, _, err := net.ParseCIDR(nexthop)
		if err != nil {
			continue
		}
		route := &networkservice.Route{
			NextHop: gw.String(),
		}
		if isIPv6(nexthop) {
			route.Prefix = "::/0"
			ipv6Nexthops = append(ipv6Nexthops, route)
		} else {
			route.Prefix = "0.0.0.0/0"
			ipv4Nexthops = append(ipv4Nexthops, route)
		}
	}
	// prepare the policies (only based on VIP address for now)
	c.connection.Context.IpContext.Policies = []*networkservice.PolicyRoute{}
	for _, vip := range vips {
		nexthops := ipv4Nexthops
		if isIPv6(vip) {
			nexthops = ipv6Nexthops
		}
		newPolicyRoute := &networkservice.PolicyRoute{
			From:   vip,
			Routes: nexthops,
		}
		c.connection.Context.IpContext.Policies = append(c.connection.Context.IpContext.Policies, newPolicyRoute)
	}

	// update the NSM connection
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             c.connection.GetId(),
			NetworkService: c.connection.GetNetworkService(),
			Mechanism:      c.connection.GetMechanism(),
			Labels:         c.connection.GetLabels(),
			Payload:        c.connection.GetPayload(),
			Context: &networkservice.ConnectionContext{
				IpContext: c.connection.GetContext().GetIpContext(),
			},
		},
	}
	connection, err := c.NetworkServiceClient.Request(ctx, request)
	if err != nil {
		c.logger.Error(err, "Updating VIPs")
		return fmt.Errorf("nsc connection update error: %w", err)
	}
	c.connection = connection

	c.logger.Info("VIPs updated", "vips", vips)
	c.vips = vips
	c.localIPs = getLocalIPs(c.connection.GetContext().GetIpContext().GetSrcIpAddrs(), vips, c.getGateways())
	return nil
}

// Equals checks if the conduit is equal to the one in parameter
func (c *Conduit) Equals(conduit *ambassadorAPI.Conduit) bool {
	return c.Conduit.Equals(conduit)
}

func (c *Conduit) isConnected() bool {
	return c.connection != nil
}

func isIPv6(address string) bool {
	return strings.Count(address, ":") >= 2
}

// TODO: Requires the IPs of the bridge
// GetDstIpAddrs doesn't work in IPv6
func (c *Conduit) getGateways() []string {
	if c.connection != nil {
		return c.connection.GetContext().GetIpContext().GetExtraPrefixes()
	}
	return []string{}
}

// monitor the current nsm connection in order to get the new local IPs in case of a change made by the proxy.
// TODO: Reflect the NSM connection status with the stream status in the TAPA API.
func (c *Conduit) monitorConnection(ctx context.Context, initialConnection *networkservice.Connection) {
	if c.MonitorConnectionClient == nil {
		return
	}
	logger := c.logger.WithValues("func", "monitorConnection", "ID", initialConnection.GetId())
	monitorScope := &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Id: initialConnection.GetId(),
			},
		},
	}
	_ = retry.Do(func() error {
		monitorConnectionsClient, err := c.MonitorConnectionClient.MonitorConnections(ctx, monitorScope)
		if err != nil {
			return fmt.Errorf("failed to create connection monitor client: %w", err)
		}
		for {
			mccResponse, err := monitorConnectionsClient.Recv()
			if err != nil {
				s, _ := grpcStatus.FromError(err)
				if s.Code() != grpcCodes.Canceled {
					// client did not close the connection
					// (refer to https://github.com/networkservicemesh/sdk/blob/v1.11.1/pkg/networkservice/common/heal/eventloop.go#L114)
					logger.Info("Connection monitor lost contact with local NSMgr")
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("connection monitor client receive error: %w", err)
			}
			for _, connection := range mccResponse.Connections {
				path := connection.GetPath()
				if path != nil && len(path.PathSegments) >= 1 && path.PathSegments[0].Id == initialConnection.GetId() {
					// Check for control plane down or connection close events to log them
					//
					// TODO: Check if it'd make sense closing the streams when the connection is
					// closed (to synchronize NSP and its consumers), and assess possible side-effects.
					// (Connection recovery would need to be tracked somehow to re-open streams.)
					if connection.GetState() == networkservice.State_DOWN || mccResponse.Type == networkservice.ConnectionEventType_DELETE {
						msg := "Connection monitor received delete event" // connection closed (e.g. due to NSM heal with reselect)
						if connection.GetState() == networkservice.State_DOWN {
							msg = "Connection monitor received down event" // control plane is down
						}
						logger.Info(msg, "event type", mccResponse.Type, "connection state", connection.GetState())
					}
					c.mu.Lock()
					// Check for changes involving localIPs of the connection
					if c.isConnected() {
						c.connection.Context = connection.GetContext()
						oldLocalIPsSet := sets.New(c.localIPs...)
						c.localIPs = getLocalIPs(c.connection.GetContext().GetIpContext().GetSrcIpAddrs(), c.vips, c.getGateways())
						if !oldLocalIPsSet.Equal(sets.New(c.localIPs...)) {
							logger.Info("Connection IPs updated, streams require update", "ID", initialConnection.GetId(), "localIPs", c.localIPs, "old localIPs", oldLocalIPsSet)
							// Trigger stream update to announce new localIPs.
							// Note: First let's close streams in the conduit to
							// release identifiers currently in use. In order to
							// avoid lingering outdated Targets, and registration
							// delays due to low availability of free identifiers.
							// Note: Locks are held, context cannot be cancelled.
							stopCtx, stopCancel := context.WithTimeout(ctx, 2*c.nspEntryTimeout) // don't risk blocking indefinitely
							if err := c.StreamManager.Stop(stopCtx); err != nil {
								logger.Info("Connection update triggered stream error", "err", err, "ID", initialConnection.GetId())
							}
							stopCancel()
							c.StreamManager.Run()
						}
					}
					c.mu.Unlock()
					break
				}
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(5*time.Second),
		retry.WithErrorIngnored())
}

// removes return addresses that are not in the VIP list and that are in one of the gateway subnet
func getLocalIPs(addresses []string, vips []string, gateways []string) []string {
	return getIPsInGatewaySubnet(getLocalIPsBasedOnVIP(addresses, vips), gateways)
}

// remove VIPs from addresses list
func getLocalIPsBasedOnVIP(addresses []string, vips []string) []string {
	res := []string{}
	vipsMap := map[string]struct{}{}
	for _, vip := range vips {
		ip, ipNet, err := net.ParseCIDR(vip)
		if err != nil {
			continue
		}
		prefixLength, _ := ipNet.Mask.Size()
		vipsMap[fmt.Sprintf("%s/%d", ip, prefixLength)] = struct{}{}
	}
	for _, address := range addresses {
		ip, ipNet, err := net.ParseCIDR(address)
		if err != nil {
			continue
		}
		prefixLength, _ := ipNet.Mask.Size()
		cidr := fmt.Sprintf("%s/%d", ip, prefixLength) // reformat in case address have been modified (e.g. IPv6 format)
		_, exists := vipsMap[cidr]
		if !exists {
			res = append(res, cidr)
		}
	}
	return res
}

// Remove all addresses that are not in subnet of gateways
func getIPsInGatewaySubnet(addresses []string, gateways []string) []string {
	res := []string{}
	subnets := []*net.IPNet{}
	for _, prefix := range gateways {
		_, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			continue
		}
		subnets = append(subnets, ipNet)
	}
	for _, prefix := range addresses {
		ip, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			continue
		}
		prefixLength, _ := ipNet.Mask.Size()
		for _, subnet := range subnets {
			subnetLength, _ := subnet.Mask.Size()
			if subnet.Contains(ip) && prefixLength >= subnetLength {
				cidr := fmt.Sprintf("%s/%d", ip, prefixLength) // reformat in case srcIpAddrs have been modified (e.g. IPv6 format)
				res = append(res, cidr)
				break
			}
		}
	}
	return res
}

func formatPrefixes(prefixes []string) []string {
	res := []string{}
	for _, prefix := range prefixes {
		ip, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			continue
		}
		prefixLength, _ := ipNet.Mask.Size()
		cidr := fmt.Sprintf("%s/%d", ip, prefixLength)
		res = append(res, cidr)
	}
	return res
}
