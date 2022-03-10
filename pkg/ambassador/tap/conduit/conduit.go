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
	"net"
	"strings"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/conduit"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
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
	NodeName             string
	NetworkServiceClient networkservice.NetworkServiceClient
	Configuration        Configuration
	StreamManager        StreamManager
	NetUtils             networking.Utils
	StreamFactory        StreamFactory
	connection           *networkservice.Connection
	mu                   sync.Mutex
	localIPs             []string
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
	streamRegistry types.Registry,
	netUtils networking.Utils) (*Conduit, error) {
	c := &Conduit{
		TargetName:           targetName,
		Namespace:            namespace,
		Conduit:              conduit,
		NodeName:             nodeName,
		NetworkServiceClient: networkServiceClient,
		NetUtils:             netUtils,
		connection:           nil,
		localIPs:             []string{},
	}
	c.StreamFactory = stream.NewFactory(targetRegistryClient, stream.MaxNumberOfTargets, c)
	c.StreamManager = NewStreamManager(configurationManagerClient, targetRegistryClient, streamRegistry, c.StreamFactory, PendingTime)
	c.Configuration = newConfigurationImpl(c.SetVIPs, c.StreamManager.SetStreams, c.Conduit.ToNSP(), configurationManagerClient)
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
	logrus.Infof("Attempt to connect conduit: %v", c.Conduit)
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
	logrus.Infof("Conduit connected: %v", c.Conduit)
	c.connection = connection
	c.localIPs = c.connection.GetContext().GetIpContext().GetSrcIpAddrs()

	c.Configuration.Watch()

	c.StreamManager.Run()
	return nil
}

// Disconnect closes the connection from NSM, closes all streams
// and stop the VIP watcher
func (c *Conduit) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	logrus.Infof("Disconnect from conduit: %v", c.Conduit)
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
	logrus.Infof("Add stream: %v to conduit: %v", strm, c.Conduit)
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

// GetStreams returns the local IPs for this conduit
func (c *Conduit) GetIPs() []string {
	if c.connection != nil {
		return c.localIPs
	}
	return []string{}
}

// SetVIPs checks the vips which has to be added or removed
func (c *Conduit) SetVIPs(vips []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isConnected() {
		return nil
	}
	// prepare SrcIpAddrs (IPs allocated by the proxy + VIPs)
	c.connection.Context.IpContext.SrcIpAddrs = append(c.GetIPs(), vips...)
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
	var err error
	// update the NSM connection
	// TODO: retry if error returned?
	c.connection, err = c.NetworkServiceClient.Request(context.TODO(),
		&networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             c.connection.GetId(),
				NetworkService: c.connection.GetNetworkService(),
				Labels:         c.connection.GetLabels(),
				Payload:        c.connection.GetPayload(),
				Context: &networkservice.ConnectionContext{
					IpContext: c.connection.GetContext().GetIpContext(),
				},
			},
			MechanismPreferences: []*networkservice.Mechanism{
				{
					Cls:  cls.LOCAL,
					Type: kernelmech.MECHANISM,
				},
			},
		})
	if err != nil {
		return fmt.Errorf("error updating the VIPs in conduit: %v - %v", c.Conduit, err)
	}
	logrus.Infof("VIPs in conduit updated: %v - %v", c.Conduit, vips)
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
