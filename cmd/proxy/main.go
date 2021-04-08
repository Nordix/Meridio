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
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/nordix/meridio/pkg/proxy"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	proxySubnet, err := getProxySubnet(config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	vip, _ := netlink.ParseAddr(config.VIP)
	if err != nil {
		logrus.Fatalf("Error Parsing the VIP: %v", err)
	}

	linkMonitor, err := networking.NewLinkMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}

	p := proxy.NewProxy(vip, proxySubnet)
	interfaceMonitorEndpoint := nsm.NewInterfaceMonitorEndpoint(p)
	proxyEndpoint := proxy.NewProxyEndpoint(p)
	linkMonitor.Subscribe(interfaceMonitorEndpoint)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)

	go startNSC(nsmAPIClient, config.NetworkServiceName, p, p)

	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	startNSE(ctx, endpointConfig, nsmAPIClient, proxyEndpoint, interfaceMonitorEndpoint, config.NSPService)
}

func getProxySubnet(config Config) (*netlink.Addr, error) {
	subnetPool, err := netlink.ParseAddr(config.SubnetPool)
	if err != nil {
		return nil, errors.Wrap(err, "Error Parsing subnet pool")
	}
	ipamClient, err := ipam.NewIpamClient(config.IPAMService)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating new ipam client")
	}
	proxySubnet, err := ipamClient.AllocateSubnet(subnetPool, 24)
	if err != nil {
		return nil, errors.Wrap(err, "Error AllocateSubnet")
	}
	return proxySubnet, nil
}

func startNSC(nsmAPIClient *nsm.APIClient,
	networkServiceName string,
	interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber,
	nscConnectionFactory client.NSCConnectionFactory) {

	monitor := client.NewMonitor(networkServiceName, nsmAPIClient, nsmAPIClient)
	monitor.SetInterfaceMonitorSubscriber(interfaceMonitorSubscriber)
	monitor.SetNSCConnectionFactory(nscConnectionFactory)
	monitor.Start()
}

func startNSE(ctx context.Context,
	config *endpoint.Config,
	nsmAPIClient *nsm.APIClient,
	proxyEndpoint *proxy.ProxyEndpoint,
	interfaceMonitorEndpoint *nsm.InterfaceMonitorEndpoint,
	nspService string) {

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		nsm.NewInterfaceNameEndpoint(),
		proxyEndpoint,
		interfaceMonitorEndpoint,
		nsp.NewNSPEndpoint(nspService),
		sendfd.NewServer(),
	}

	ep, err := endpoint.NewEndpoint(ctx, config, nsmAPIClient.NetworkServiceRegistryClient, nsmAPIClient.NetworkServiceEndpointRegistryClient)
	if err != nil {
		logrus.Fatalf("unable to create a new nse %+v", err)
	}

	err = ep.Start(responderEndpoint...)
	if err != nil {
		logrus.Errorf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	<-ctx.Done()
}
