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
	"github.com/vishvananda/netlink"
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

	_, err = netlink.ParseAddr(config.VIP)
	if err != nil {
		logrus.Fatalf("Error Parsing the VIP: %v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}
	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}

	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	st := NewSimpleTarget(identifier, config.VIP, netUtils)

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
	client := client.NewNetworkServiceClient(clientConfig, apiClient.GRPCClient, networkServiceClient)
	err = client.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             fmt.Sprintf("%s-%s-%d", config.Name, config.ProxyNetworkServiceName, 0),
			NetworkService: config.ProxyNetworkServiceName,
			Labels:         map[string]string{"forwarder": "forwarder-vpp"},
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
	identifier       int
	sourceBasedRoute networking.SourceBasedRoute
	vip              string
}

// InterfaceCreated -
func (st *SimpleTarget) InterfaceCreated(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Adding nexthop: no gateway: %v", intf)
		return
	}
	gateway := intf.GetGatewayPrefixes()[0]
	err := st.sourceBasedRoute.AddNexthop(gateway)
	if err != nil {
		logrus.Errorf("SimpleTarget: Adding nexthop (%v) to source base route err: %v", gateway, err)
	}
}

// InterfaceDeleted -
func (st *SimpleTarget) InterfaceDeleted(intf networking.Iface) {
	if len(intf.GetGatewayPrefixes()) <= 0 {
		logrus.Errorf("SimpleTarget: Removing nexthop: no gateway: %v", intf)
		return
	}
	gateway := intf.GetGatewayPrefixes()[0]
	err := st.sourceBasedRoute.RemoveNexthop(gateway)
	if err != nil {
		logrus.Errorf("SimpleTarget: Removing nexthop (%v) from source base route err: %v", gateway, err)
	}
}

func NewSimpleTarget(identifier int, vip string, netUtils networking.Utils) *SimpleTarget {
	sourceBasedRoute, err := netUtils.NewSourceBasedRoute(10, vip)
	if err != nil {
		logrus.Errorf("SimpleTarget: NewSourceBasedRoute err: %v", err)
	}
	err = netUtils.AddVIP(vip)
	if err != nil {
		logrus.Errorf("SimpleTarget: err AddVIP: %v", err)
	}
	simpleTarget := &SimpleTarget{
		identifier:       identifier,
		sourceBasedRoute: sourceBasedRoute,
		vip:              vip,
	}
	return simpleTarget
}
