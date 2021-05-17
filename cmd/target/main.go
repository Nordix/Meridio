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

	<-ctx.Done()
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
	identifier        int
	sourceBasedRoutes []networking.SourceBasedRoute
	vips              []string
	netUtils          networking.Utils
}

// InterfaceCreated -
func (st *SimpleTarget) InterfaceCreated(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Adding nexthop: no gateway: %v", intf)
		return
	}
	for _, gateway := range intf.GetGatewayPrefixes() {
		for _, sourceBasedRoute := range st.sourceBasedRoutes {
			err := sourceBasedRoute.AddNexthop(gateway)
			if err != nil {
				logrus.Errorf("SimpleTarget: Adding nexthop (%v) to source base route err: %v", gateway, err)
			}
		}
	}
}

// InterfaceDeleted -
func (st *SimpleTarget) InterfaceDeleted(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Removing nexthop: no gateway: %v", intf)
		return
	}

	for _, gateway := range intf.GetGatewayPrefixes() {
		for _, sourceBasedRoute := range st.sourceBasedRoutes {
			err := sourceBasedRoute.RemoveNexthop(gateway)
			if err != nil {
				logrus.Errorf("SimpleTarget: Removing nexthop (%v) from source base route err: %v", gateway, err)
			}
		}
	}
}

func (st *SimpleTarget) createSourceBaseRoutes() error {
	for index, vip := range st.vips {
		sourceBasedRoute, err := st.netUtils.NewSourceBasedRoute(index, vip)
		if err != nil {
			logrus.Errorf("Proxy: Error creating sourceBasedRoute: %v", err)
			return err
		}
		st.sourceBasedRoutes = append(st.sourceBasedRoutes, sourceBasedRoute)
	}
	return nil
}

func (st *SimpleTarget) addVIPs() error {
	for _, vip := range st.vips {
		err := st.netUtils.AddVIP(vip)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewSimpleTarget(identifier int, vips []string, netUtils networking.Utils) *SimpleTarget {
	simpleTarget := &SimpleTarget{
		identifier:        identifier,
		sourceBasedRoutes: []networking.SourceBasedRoute{},
		vips:              vips,
		netUtils:          netUtils,
	}
	err := simpleTarget.addVIPs()
	if err != nil {
		logrus.Errorf("SimpleTarget: err addVIPs: %v", err)
	}
	err = simpleTarget.createSourceBaseRoutes()
	if err != nil {
		logrus.Errorf("SimpleTarget: createSourceBaseRoutes err: %v", err)
	}
	return simpleTarget
}
