package target

import (
	"fmt"
	"os"
	"strconv"

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
	"google.golang.org/grpc"
)

// Connection -
type Connection struct {
	identifier           int
	networkServiceName   string
	trench               string
	networkServiceClient client.NetworkServiceClient
	vips                 []*virtualIP
	netUtils             networking.Utils
	nexthops             []string
	tableID              int
}

func (c *Connection) Delete() {
	c.deleteVIPs(c.vips)
	c.nexthops = []string{}
	c.tableID = 1
}

func (c *Connection) getAdditionalFunctionalities() networkservice.NetworkServiceClient {
	interfaceMonitor, err := c.netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, c, c.netUtils)
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

func (c *Connection) Request(cc grpc.ClientConnInterface, config *client.Config) {
	proxyNetworkServiceName := fmt.Sprintf("proxy.%s", c.networkServiceName)
	if c.trench != "" {
		proxyNetworkServiceName = fmt.Sprintf("%s.%s", proxyNetworkServiceName, c.trench)
	}
	clientConfig := &client.Config{
		Name:           config.Name,
		RequestTimeout: config.RequestTimeout,
	}
	c.networkServiceClient = client.NewSimpleNetworkServiceClient(clientConfig, cc, c.getAdditionalFunctionalities())
	err := c.networkServiceClient.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             fmt.Sprintf("%s-%s-%d", config.Name, proxyNetworkServiceName, 0),
			NetworkService: proxyNetworkServiceName,
			Labels:         make(map[string]string),
			Payload:        payload.Ethernet,
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{
					"identifier": strconv.Itoa(c.identifier),
				},
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
		logrus.Fatalf("Request err: %+v", err)
	}
}

func (c *Connection) Close() {
	c.Delete()
	err := c.networkServiceClient.Close()
	if err != nil {
		logrus.Fatalf("Close err: %+v", err)
	}
}

func (c *Connection) deleteVIPs(vips []*virtualIP) {
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
func (c *Connection) InterfaceCreated(intf networking.Iface) {
	logrus.Infof("Client: InterfaceCreated: %v", intf)
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
func (c *Connection) InterfaceDeleted(intf networking.Iface) {
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

func (c *Connection) GetVIPs() []string {
	vips := []string{}
	for _, vip := range c.vips {
		vips = append(vips, vip.prefix)
	}
	return vips
}

func (c *Connection) SetVIPs(vips []string) {
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range c.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, c.tableID, c.netUtils)
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

func NewConnection(networkServiceName string, trench string, netUtils networking.Utils) *Connection {
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	connection := &Connection{
		identifier:         identifier,
		networkServiceName: networkServiceName,
		trench:             trench,
		vips:               []*virtualIP{},
		netUtils:           netUtils,
		nexthops:           []string{},
		tableID:            1,
	}
	return connection
}
