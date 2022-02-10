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
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	sriovtoken "github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/target/types"
	"github.com/sirupsen/logrus"
)

type Conduit struct {
	Name                 string
	Trench               types.Trench
	NodeName             string
	Configuration        *Configuration
	networkServiceClient client.NetworkServiceClient
	EventChan            chan<- struct{}
	NetUtils             networking.Utils
	apiClient            *nsm.APIClient
	nsmConfig            *nsm.Config
	vips                 []*virtualIP
	nexthops             []string
	ips                  []string
	tableID              int
	streams              []types.Stream
	mu                   sync.Mutex
	status               types.ConduitStatus
}

func New(
	ctx context.Context,
	name string,
	trench types.Trench,
	nodeName string,
	apiClient *nsm.APIClient,
	nsmConfig *nsm.Config,
	eventChan chan<- struct{},
	netUtils networking.Utils) (types.Conduit, error) {
	conduit := &Conduit{
		Name:      name,
		Trench:    trench,
		NodeName:  nodeName,
		apiClient: apiClient,
		nsmConfig: nsmConfig,
		vips:      []*virtualIP{},
		nexthops:  []string{},
		ips:       []string{},
		tableID:   1,
		EventChan: eventChan,
		NetUtils:  netUtils,
		status:    types.Disconnected,
	}
	conduit.Configuration = NewConfiguration(conduit, trench.GetConfigurationManagerClient())
	err := trench.AddConduit(ctx, conduit)
	if err != nil {
		return nil, err
	}
	return conduit, nil
}

func (c *Conduit) GetName() string {
	return c.Name
}

func (c *Conduit) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	proxyNetworkServiceName := c.getNetworkServiceName()
	clientConfig := &client.Config{
		Name:           c.nsmConfig.Name,
		RequestTimeout: c.nsmConfig.RequestTimeout,
		ConnectTo:      c.nsmConfig.ConnectTo,
	}

	nscCtx := context.Background()
	nscCtx = log.WithLog(nscCtx, logruslogger.New(nscCtx)) // allow NSM logs
	c.networkServiceClient = client.NewSimpleNetworkServiceClient(nscCtx, clientConfig, c.apiClient, c.getAdditionalFunctionalities(nscCtx))
	err := c.networkServiceClient.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             fmt.Sprintf("%s-%s-%d", c.nsmConfig.Name, proxyNetworkServiceName, 0),
			NetworkService: proxyNetworkServiceName,
			Labels: map[string]string{
				"nodeName": c.NodeName,
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
	c.status = types.Connected
	c.notifyWatcher()
	go c.Configuration.WatchVIPs(context.Background())
	return nil
}

func (c *Conduit) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, stream := range c.streams {
		err := stream.Close(ctx)
		if err != nil {
			return err
		}
	}
	err := c.networkServiceClient.Close()
	if err != nil {
		return err
	}
	c.Configuration.Delete() // todo: https://github.com/Nordix/Meridio/pull/139#discussion_r788055463
	c.status = types.Disconnected
	c.notifyWatcher()
	c.deleteVIPs(c.vips)
	c.nexthops = []string{}
	c.tableID = 1
	return nil
}

func (c *Conduit) AddStream(ctx context.Context, stream types.Stream) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	index := c.getIndex(stream.GetName())
	if index >= 0 {
		return errors.New("this stream is already opened")
	}
	err := stream.Open(ctx)
	if err != nil {
		return err
	}
	c.streams = append(c.streams, stream)
	return nil
}

func (c *Conduit) RemoveStream(ctx context.Context, stream types.Stream) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	index := c.getIndex(stream.GetName())
	if index < 0 {
		return errors.New("this stream is not opened")
	}
	err := stream.Close(ctx)
	if err != nil {
		return err
	}
	c.streams = append(c.streams[:index], c.streams[index+1:]...)
	return nil
}

func (c *Conduit) GetStreams(stream *nspAPI.Stream) []types.Stream {
	c.mu.Lock()
	defer c.mu.Unlock()
	if stream == nil {
		return c.streams
	}
	streams := []types.Stream{}
	for _, s := range c.streams {
		if s.GetStatus() == types.Closed || !s.Equals(stream) {
			continue
		}
		streams = append(streams, s)
	}
	return streams
}

func (c *Conduit) GetTrench() types.Trench {
	return c.Trench
}

func (c *Conduit) GetIPs() []string {
	return c.ips
}

func (c *Conduit) SetVIPs(vips []string) error {
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
			for _, nexthop := range c.nexthops {
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

func (c *Conduit) Equals(conduit *nspAPI.Conduit) bool {
	if conduit == nil {
		return true
	}
	name := true
	if conduit.GetName() != "" {
		name = c.GetName() == conduit.GetName()
	}
	return name && c.GetTrench().Equals(conduit.GetTrench())
}

func (c *Conduit) GetStatus() types.ConduitStatus {
	return c.status
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

func (c *Conduit) getIndex(streamName string) int {
	for i, stream := range c.streams {
		if stream.GetName() == streamName {
			return i
		}
	}
	return -1
}

func (c *Conduit) notifyWatcher() {
	if c.EventChan == nil {
		return
	}
	c.EventChan <- struct{}{}
}

func (c *Conduit) getNetworkServiceName() string {
	return fmt.Sprintf("%s.%s.%s.%s", proxyPrefix, c.GetName(), c.GetTrench().GetName(), c.GetTrench().GetNamespace())
}

func (c *Conduit) getAdditionalFunctionalities(ctx context.Context) networkservice.NetworkServiceClient {
	interfaceMonitor, err := c.NetUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, c, c.NetUtils)
	// Note: tell NSM to use "nsc" for the interface name
	// Must be revisited once multiple NSM client interface are to be supported on the application side.
	additionalFunctionalities := chain.NewNetworkServiceClient(
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient(kernel.WithInterfaceName("nsc"))),
		}),
		interfaceMonitorClient,
	)
	return additionalFunctionalities
}

func (c *Conduit) InterfaceCreated(intf networking.Iface) {
	logrus.Infof("Client: InterfaceCreated: %v", intf)
	c.ips = intf.GetLocalPrefixes()
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("Client: Adding nexthop: no gateway: %v", intf)
		return
	}
	for _, gateway := range intf.GetGatewayPrefixes() {
		for _, vip := range c.vips {
			err := vip.AddNexthop(gateway)
			if err != nil {
				logrus.Errorf("Client: Adding nexthop (%v) to source base route err: %v", gateway, err)
			}
		}
		c.nexthops = append(c.nexthops, gateway)
	}
}

func (c *Conduit) InterfaceDeleted(intf networking.Iface) {
	c.ips = []string{}
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("Client: Removing nexthop: no gateway: %v", intf)
		return
	}
	for _, gateway := range intf.GetGatewayPrefixes() {
		for _, vip := range c.vips {
			err := vip.RemoveNexthop(gateway)
			if err != nil {
				logrus.Errorf("Client: Removing nexthop (%v) from source base route err: %v", gateway, err)
			}
		}
		for index, nexthop := range c.nexthops {
			if nexthop == gateway {
				c.nexthops = append(c.nexthops[:index], c.nexthops[index+1:]...)
			}
		}
	}
}
