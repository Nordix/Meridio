package target

import (
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	sriovtoken "github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/sirupsen/logrus"
)

// Conduit -
type Conduit struct {
	name                 string
	trench               *Trench
	networkServiceClient client.NetworkServiceClient
	vips                 []*virtualIP
	nexthops             []string
	ips                  []string
	tableID              int
	stream               *Stream
}

func (c *Conduit) Delete() error {
	err := c.DeleteStream()
	if err != nil {
		return err
	}
	err = c.networkServiceClient.Close()
	if err != nil {
		return err
	}
	c.deleteVIPs(c.vips)
	c.nexthops = []string{}
	c.tableID = 1
	return nil
}

func (c *Conduit) RequestStream() error {
	c.stream = NewStream(c.name, c)
	return c.stream.Request()
}

func (c *Conduit) DeleteStream() error {
	if c.stream == nil {
		return nil
	}
	err := c.stream.Delete()
	if err != nil {
		return err
	}
	c.stream = nil
	return nil
}

func (c *Conduit) getAdditionalFunctionalities() networkservice.NetworkServiceClient {
	interfaceMonitor, err := c.GetConfig().netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, c, c.GetConfig().netUtils)
	additionalFunctionalities := chain.NewNetworkServiceClient(
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		interfacename.NewClient("nsc", &interfacename.RandomGenerator{}),
		interfaceMonitorClient,
		sendfd.NewClient(),
	)
	return additionalFunctionalities
}

func (c *Conduit) request() error {
	proxyNetworkServiceName := c.GetNetworkServiceName()
	clientConfig := &client.Config{
		Name:           c.GetConfig().nsmConfig.Name,
		RequestTimeout: c.GetConfig().nsmConfig.RequestTimeout,
	}
	c.networkServiceClient = client.NewSimpleNetworkServiceClient(clientConfig, c.GetConfig().apiClient.GRPCClient, c.getAdditionalFunctionalities())
	err := c.networkServiceClient.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             fmt.Sprintf("%s-%s-%d", c.GetConfig().nsmConfig.Name, proxyNetworkServiceName, 0),
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
	return err
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
				logrus.Errorf("Client: Error deleting vip: %v", err)
			}
		}
	}
}

// InterfaceCreated -
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

// InterfaceDeleted -
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

func (c *Conduit) GetVIPs() []string {
	vips := []string{}
	for _, vip := range c.vips {
		vips = append(vips, vip.prefix)
	}
	return vips
}

func (c *Conduit) SetVIPs(vips []string) {
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range c.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, c.tableID, c.GetConfig().netUtils)
			if err != nil {
				logrus.Errorf("SimpleTarget: Error adding SourceBaseRoute: %v", err)
				continue
			}
			c.tableID++
			c.vips = append(c.vips, newVIP)
			for _, nexthop := range c.nexthops {
				err = newVIP.AddNexthop(nexthop)
				if err != nil {
					logrus.Errorf("Client: Error adding nexthop: %v", err)
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
}

func (c *Conduit) GetNetworkServiceName() string {
	return fmt.Sprintf("proxy.%s.%s.%s", c.name, c.GetTrenchName(), c.GetNamespace())
}

func (c *Conduit) GetName() string {
	return c.name
}

func (c *Conduit) GetTrenchName() string {
	return c.trench.GetName()
}

func (c *Conduit) GetNamespace() string {
	return c.trench.GetNamespace()
}

func (c *Conduit) GetConfig() *Config {
	return c.trench.GetConfig()
}

func NewConduit(name string, trench *Trench) (*Conduit, error) {
	conduit := &Conduit{
		name:     name,
		trench:   trench,
		vips:     []*virtualIP{},
		nexthops: []string{},
		ips:      []string{},
		tableID:  1,
	}
	err := conduit.request()
	return conduit, err
}
