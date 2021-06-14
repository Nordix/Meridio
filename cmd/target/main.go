package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kelseyhightower/envconfig"
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
	"github.com/nordix/meridio/pkg/configuration"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}
	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}

	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	st := NewSimpleTarget(identifier, config.VIPs, netUtils)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	apiClient := nsm.NewAPIClient(ctx, apiClientConfig)

	extraContext := make(map[string]string)
	extraContext["identifier"] = strconv.Itoa(st.identifier)

	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, st, netUtils)

	labels := map[string]string{"forwarder": "forwarder-vpp"}
	if config.Host != "" {
		labels["host"] = config.Host
	}

	networkServiceClient := chain.NewNetworkServiceClient(
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		interfacename.NewClient("nsc", &interfacename.RandomGenerator{}),
		interfaceMonitorClient,
		sendfd.NewClient(),
	)
	clientConfig := &client.Config{
		Name:           config.Name,
		RequestTimeout: config.RequestTimeout,
	}
	client := client.NewSimpleNetworkServiceClient(clientConfig, apiClient.GRPCClient, networkServiceClient)
	err = client.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             fmt.Sprintf("%s-%s-%d", config.Name, config.ProxyNetworkServiceName, 0),
			NetworkService: config.ProxyNetworkServiceName,
			Labels:         labels,
			Payload:        payload.Ethernet,
			Context: &networkservice.ConnectionContext{
				ExtraContext: extraContext,
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
		logrus.Fatalf("client.Request err: %+v", err)
	}
	defer client.Close()

	configWatcher := make(chan *configuration.Config)
	configurationWatcher := configuration.NewWatcher(config.ConfigMapName, config.Namespace, configWatcher)
	go configurationWatcher.Start()

	for {
		select {
		case config := <-configWatcher:
			st.SetVIPs(config.VIPs)
		case <-ctx.Done():
			return
		}
	}
}

// Hash -
func Hash(s string, n int) int {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		logrus.Fatalf("error hashing: %+v", err)
	}
	return int(h.Sum32())%n + 1
}

// SimpleTarget -
type SimpleTarget struct {
	identifier int
	vips       []*virtualIP
	netUtils   networking.Utils
	nexthops   []string
	tableID    int
}

// InterfaceCreated -
func (st *SimpleTarget) InterfaceCreated(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Adding nexthop: no gateway: %v", intf)
		return
	}
	for _, gateway := range intf.GetGatewayPrefixes() {
		for _, vip := range st.vips {
			err := vip.AddNexthop(gateway)
			if err != nil {
				logrus.Errorf("SimpleTarget: Adding nexthop (%v) to source base route err: %v", gateway, err)
			}
		}
		st.nexthops = append(st.nexthops, gateway)
	}
}

// InterfaceDeleted -
func (st *SimpleTarget) InterfaceDeleted(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Removing nexthop: no gateway: %v", intf)
		return
	}

	for _, gateway := range intf.GetGatewayPrefixes() {
		for _, vip := range st.vips {
			err := vip.RemoveNexthop(gateway)
			if err != nil {
				logrus.Errorf("SimpleTarget: Removing nexthop (%v) from source base route err: %v", gateway, err)
			}
		}
		for index, nexthop := range st.nexthops {
			if nexthop == gateway {
				st.nexthops = append(st.nexthops[:index], st.nexthops[index+1:]...)
			}
		}
	}
}

func (st *SimpleTarget) SetVIPs(vips []string) {
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range st.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, st.tableID, st.netUtils)
			if err != nil {
				logrus.Errorf("SimpleTarget: Error adding SourceBaseRoute: %v", err)
				continue
			}
			st.tableID++
			st.vips = append(st.vips, newVIP)
			for _, nexthop := range st.nexthops {
				err = newVIP.AddNexthop(nexthop)
				if err != nil {
					logrus.Errorf("SimpleTarget: Error adding nexthop: %v", err)
				}
			}
		}
		delete(currentVIPs, vip)
	}
	// delete remaining vips
	for index := 0; index < len(st.vips); index++ {
		vip := st.vips[index]
		if _, ok := currentVIPs[vip.prefix]; ok {
			st.vips = append(st.vips[:index], st.vips[index+1:]...)
			index--
			err := vip.Delete()
			if err != nil {
				logrus.Errorf("SimpleTarget: Error deleting vip: %v", err)
			}
		}
	}
}

func NewSimpleTarget(identifier int, vips []string, netUtils networking.Utils) *SimpleTarget {
	simpleTarget := &SimpleTarget{
		identifier: identifier,
		vips:       []*virtualIP{},
		netUtils:   netUtils,
		nexthops:   []string{},
		tableID:    1,
	}
	simpleTarget.SetVIPs(vips)
	return simpleTarget
}

type virtualIP struct {
	sourceBasedRoute networking.SourceBasedRoute
	prefix           string
	netUtils         networking.Utils
}

func (vip *virtualIP) Delete() error {
	err := vip.netUtils.DeleteVIP(vip.prefix)
	if err != nil {
		return err
	}
	return vip.removeSourceBaseRoute()
}

func (vip *virtualIP) AddNexthop(ip string) error {
	return vip.sourceBasedRoute.AddNexthop(ip)
}

func (vip *virtualIP) RemoveNexthop(ip string) error {
	return vip.sourceBasedRoute.RemoveNexthop(ip)
}

func (vip *virtualIP) createSourceBaseRoute(tableID int) error {
	var err error
	vip.sourceBasedRoute, err = vip.netUtils.NewSourceBasedRoute(tableID, vip.prefix)
	logrus.Infof("VIP Simple target: sourceBasedRoute index - vip: %v - %v", tableID, vip.prefix)
	if err != nil {
		return err
	}
	return nil
}

func (vip *virtualIP) removeSourceBaseRoute() error {
	return vip.sourceBasedRoute.Delete()
}

func newVirtualIP(prefix string, tableID int, netUtils networking.Utils) (*virtualIP, error) {
	vip := &virtualIP{
		prefix:   prefix,
		netUtils: netUtils,
	}
	err := vip.createSourceBaseRoute(tableID)
	if err != nil {
		return nil, err
	}
	err = vip.netUtils.AddVIP(vip.prefix)
	if err != nil {
		return nil, err
	}
	return vip, nil
}
