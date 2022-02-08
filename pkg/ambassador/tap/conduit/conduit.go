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
	vips                 []*virtualIP
	tableID              int
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
		vips:                 []*virtualIP{},
		tableID:              1,
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
	// reset the VIPs related configuration
	c.deleteVIPs(c.vips)
	c.tableID = 1
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
		return c.connection.GetContext().GetIpContext().GetSrcIpAddrs()
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
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range c.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, c.tableID, c.NetUtils)
			if err != nil {
				logrus.Errorf("SimpleTarget: Error adding SourceBaseRoute: %v", err) // todo: err handling
				continue
			}
			c.tableID++
			c.vips = append(c.vips, newVIP)
			for _, nexthop := range c.getGateways() {
				err = newVIP.AddNexthop(nexthop)
				if err != nil {
					logrus.Errorf("Client: Error adding nexthop: %v", err) // todo: err handling
				}
			}
		}
		delete(currentVIPs, vip)
	}
	// delete remaining vips
	vipsSlice := []*virtualIP{}
	for _, vip := range currentVIPs {
		vipsSlice = append(vipsSlice, vip)
	}
	c.deleteVIPs(vipsSlice)
	return nil
}

// Equals checks if the conduit is equal to the one in parameter
func (c *Conduit) Equals(conduit *ambassadorAPI.Conduit) bool {
	return c.Conduit.Equals(conduit)
}

func (c *Conduit) isConnected() bool {
	return c.connection != nil
}

func (c *Conduit) deleteVIPs(vips []*virtualIP) {
	vipsMap := make(map[string]*virtualIP)
	for _, vip := range vips {
		vipsMap[vip.prefix] = vip
	}
	for index := 0; index < len(c.vips); index++ {
		vip := c.vips[index]
		if _, ok := vipsMap[vip.prefix]; ok {
			c.vips = append(c.vips[:index], c.vips[index+1:]...)
			index--
			err := vip.Delete()
			if err != nil {
				logrus.Errorf("Client: Error deleting vip: %v", err) // todo: err handling
			}
		}
	}
}

// TODO: Requires the IPs of the bridge
// GetDstIpAddrs doesn't work in IPv6
func (c *Conduit) getGateways() []string {
	if c.connection != nil {
		return c.connection.GetContext().GetIpContext().GetExtraPrefixes()
	}
	return []string{}
}
