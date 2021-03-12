package main

import (
	"context"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/proxy"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	subnetPool, err := netlink.ParseAddr("169.255.0.0/16")
	if err != nil {
		logrus.Errorf("Error Parsing subnet pool: %+v", err)
	}

	ipamServiceIPPort := "ipam-service:7777"
	ipamClient, err := ipam.NewIpamClient(ipamServiceIPPort)
	if err != nil {
		logrus.Errorf("Error creating New Ipam Client: %+v", err)
	}
	proxySubnet, err := ipamClient.AllocateSubnet(subnetPool, 24)
	if err != nil {
		logrus.Errorf("Error AllocateSubnet: %+v", err)
	}

	// ********************************************************************************
	// Start the Proxy (NSE + NSC)
	// ********************************************************************************

	vip, _ := netlink.ParseAddr("20.0.0.1/32")

	linkMonitor, err := networking.NewLinkMonitor()
	if err != nil {
		logrus.Errorf("Error creating link monitor: %+v", err)
	}
	p := proxy.NewProxy(vip, proxySubnet)
	interfaceMonitorEndpoint := nsm.NewInterfaceMonitorEndpoint(p)
	proxyEndpoint := proxy.NewProxyEndpoint(p)
	linkMonitor.Subscribe(interfaceMonitorEndpoint)

	go StartNSC(ctx, p, p)
	StartNSE(ctx, proxyEndpoint, interfaceMonitorEndpoint)
}

// StartNSC -
func StartNSC(ctx context.Context, interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, nscConnectionFactory client.NSCConnectionFactory) {
	rootConf := &client.Config{}
	if err := envconfig.Usage("nsm", rootConf); err != nil {
		logrus.Errorf("%+v", err)
	}
	if err := envconfig.Process("nsm", rootConf); err != nil {
		logrus.Errorf("error processing rootConf from env: %+v", err)
	}
	logrus.Infof("rootConf: %+v", rootConf)

	apiClient := nsm.NewAPIClient(ctx, rootConf)
	monitor := client.NewMonitor("load-balancer", apiClient, apiClient)
	monitor.SetInterfaceMonitorSubscriber(interfaceMonitorSubscriber)
	monitor.SetNSCConnectionFactory(nscConnectionFactory)
	monitor.Start()
}

// StartNSE -
func StartNSE(ctx context.Context, proxyEndpoint *proxy.ProxyEndpoint, interfaceMonitorEndpoint *nsm.InterfaceMonitorEndpoint) {
	// get config from environment
	config := new(endpoint.Config)
	if err := config.Process(); err != nil {
		logrus.Fatal(err.Error())
	}

	logrus.Infof("Config: %#v", config)

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		proxyEndpoint,
		interfaceMonitorEndpoint,
		sendfd.NewServer(),
	}

	ep := endpoint.NewEndpoint(ctx, config)

	err := ep.Start(responderEndpoint...)
	if err != nil {
		logrus.Errorf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	<-ctx.Done()
}
