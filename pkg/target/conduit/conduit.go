/*
Copyright (c) 2021 Nordix Foundation

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
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	targetAPI "github.com/nordix/meridio/api/target"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/nordix/meridio/pkg/target/types"
	"github.com/sirupsen/logrus"
)

type Conduit struct {
	Name                 string
	Trench               types.Trench
	networkServiceClient client.NetworkServiceClient
	ConduitWatcher       chan<- *targetAPI.ConduitEvent
	NetUtils             networking.Utils
	apiClient            *nsm.APIClient
	nsmConfig            *nsm.Config
	vips                 []*virtualIP
	nexthops             []string
	ips                  []string
	tableID              int
	streams              []types.Stream
	mu                   sync.Mutex
}

func New(
	ctx context.Context,
	name string,
	trench types.Trench,
	apiClient *nsm.APIClient,
	nsmConfig *nsm.Config,
	conduitWatcher chan<- *targetAPI.ConduitEvent,
	netUtils networking.Utils) (types.Conduit, error) {
	conduit := &Conduit{
		Name:           name,
		Trench:         trench,
		apiClient:      apiClient,
		nsmConfig:      nsmConfig,
		vips:           []*virtualIP{},
		nexthops:       []string{},
		ips:            []string{},
		tableID:        1,
		ConduitWatcher: conduitWatcher,
		NetUtils:       netUtils,
	}
	return conduit, trench.AddConduit(ctx, conduit)
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
	c.networkServiceClient = client.NewSimpleNetworkServiceClient(clientConfig, c.apiClient, c.getAdditionalFunctionalities(ctx))
	err := c.networkServiceClient.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             fmt.Sprintf("%s-%s-%d", c.nsmConfig.Name, proxyNetworkServiceName, 0),
			NetworkService: proxyNetworkServiceName,
			Labels:         make(map[string]string),
			Payload:        payload.Ethernet,
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
	c.notifyWatcher(targetAPI.ConduitEventStatus_Connect)
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
	c.notifyWatcher(targetAPI.ConduitEventStatus_Disconnect)
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
	err := stream.Request(ctx)
	if err != nil {
		return nil
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
		return nil
	}
	c.streams = append(c.streams[:index], c.streams[index+1:]...)
	return nil
}

func (c *Conduit) GetStream(streamName string) types.Stream {
	c.mu.Lock()
	defer c.mu.Unlock()
	index := c.getIndex(streamName)
	if index < 0 {
		return nil
	}
	return c.streams[index]
}

func (c *Conduit) GetStreams() []types.Stream {
	return c.streams
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

func (c *Conduit) notifyWatcher(status targetAPI.ConduitEventStatus) {
	if c.ConduitWatcher == nil {
		return
	}
	c.ConduitWatcher <- &targetAPI.ConduitEvent{
		Conduit: &targetAPI.Conduit{
			NetworkServiceName: c.GetName(),
			Trench: &targetAPI.Trench{
				Name: c.GetTrench().GetName(),
			},
		},
		ConduitEventStatus: status,
	}
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
	additionalFunctionalities := chain.NewNetworkServiceClient(
		updatepath.NewClient(c.apiClient.Config.Name),
		serialize.NewClient(),
		refresh.NewClient(ctx),
		metadata.NewClient(),
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		interfacename.NewClient("nsc", &interfacename.RandomGenerator{}),
		interfaceMonitorClient,
		authorize.NewClient(),
		sendfd.NewClient(),
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
