/*
Copyright (c) 2021-2022 Nordix Foundation

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
	// RetryDelay corresponds to the time between each Request call attempt
	RetryDelay              time.Duration
	StreamFactory           StreamFactory
	connection              *networkservice.Connection
	mu                      sync.Mutex
	localIPs                []string
	vips                    []string
	logger                  logr.Logger
	monitorConnectionCancel context.CancelFunc
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
	c := &Conduit{
		TargetName:              targetName,
		Namespace:               namespace,
		Conduit:                 conduit,
		NodeName:                nodeName,
		NetworkServiceClient:    networkServiceClient,
		MonitorConnectionClient: monitorConnectionClient,
		NetUtils:                netUtils,
		RetryDelay:              1 * time.Second,
		connection:              nil,
		localIPs:                []string{},
		vips:                    []string{},
	}
	c.StreamFactory = stream.NewFactory(targetRegistryClient, c)
	c.StreamManager = NewStreamManager(configurationManagerClient, targetRegistryClient, streamRegistry, c.StreamFactory, PendingTime, nspEntryTimeout)
	c.Configuration = newConfigurationImpl(c.StreamManager.SetStreams, c.Conduit.ToNSP(), configurationManagerClient)
	c.logger = log.Logger.WithValues("class", "Conduit", "instance", conduit.Name)
	return c, nil
}

// Connect requests the connection to NSM and, if success, will open all streams added
// and watch the VIPs
func (c *Conduit) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isConnected() {
		return nil
	}
	c.logger.Info("Connect")
	nsName := conduit.GetNetworkServiceNameWithProxy(c.Conduit.GetName(), c.Conduit.GetTrench().GetName(), c.Namespace)
	connection, err := c.NetworkServiceClient.Request(ctx,
		&networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             fmt.Sprintf("%s-%s-%d", c.TargetName, nsName, 0),
				NetworkService: nsName,
				Labels: map[string]string{
					"nodeName": c.NodeName, // required to connect to the proxy on same node
				},
				Payload: payload.Ethernet,
			},
			MechanismPreferences: []*networkservice.Mechanism{
				{
					Cls:  cls.LOCAL,
					Type: kernelmech.MECHANISM,
				},
			},
		})
	if err != nil {
		return err
	}
	c.logger.Info("Connected", "connection", connection)
	c.connection = connection
	c.localIPs = c.connection.GetContext().GetIpContext().GetSrcIpAddrs()

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
	_, err := c.NetworkServiceClient.Close(ctx, c.connection)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
	}
	c.connection = nil
	return errFinal
}

// AddStream creates a stream based on its factory and will open it (in another goroutine)
func (c *Conduit) AddStream(ctx context.Context, strm *ambassadorAPI.Stream) error {
	c.logger.Info("AddStream", "stream", strm)
	if !c.Equals(strm.GetConduit()) {
		return errors.New("invalid stream for this conduit")
	}
	return c.StreamManager.AddStream(strm)
}

// RemoveStream closes and removes the stream (if existing), and removes it from the
// stream registry.
func (c *Conduit) RemoveStream(ctx context.Context, strm *ambassadorAPI.Stream) error {
	return c.StreamManager.RemoveStream(ctx, strm)
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
	// TODO: remove VIPs overlapping with internal subnets.
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
	_ = retry.Do(func() error {
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second) // todo: configurable timeout
		defer cancel()
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
			return err
		}
		c.connection = connection
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(c.RetryDelay))
	c.logger.Info("VIPs updated", "vips", vips)
	c.vips = vips
	c.localIPs = getLocalIPs(c.connection.GetContext().GetIpContext().GetSrcIpAddrs(), vips)
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
			return err
		}
		for {
			mccResponse, err := monitorConnectionsClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			for _, connection := range mccResponse.Connections {
				path := connection.GetPath()
				if path != nil && len(path.PathSegments) >= 1 && path.PathSegments[0].Id == initialConnection.GetId() {
					c.mu.Lock()
					if c.isConnected() {
						c.connection.Context = connection.GetContext()
						c.localIPs = getLocalIPs(c.connection.GetContext().GetIpContext().GetSrcIpAddrs(), c.vips)
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

// TODO: verify the IPs are in the same subnet as ExtraPrefixes / DstIpAddrs
func getLocalIPs(srcIpAddrs []string, vips []string) []string {
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
	for _, srcIpAddr := range srcIpAddrs {
		ip, ipNet, err := net.ParseCIDR(srcIpAddr)
		if err != nil {
			continue
		}
		prefixLength, _ := ipNet.Mask.Size()
		cidr := fmt.Sprintf("%s/%d", ip, prefixLength) // reformat in case srcIpAddrs have been modified (e.g. IPv6 format)
		_, exists := vipsMap[cidr]
		if !exists {
			res = append(res, cidr)
		}
	}
	return res
}
