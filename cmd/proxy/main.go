package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	sriovtoken "github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/configuration"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/ipam"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/nordix/meridio/pkg/nsm/ipcontext"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/nordix/meridio/pkg/proxy"
	"github.com/pkg/errors"
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

	proxySubnets, err := getProxySubnets(config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	netUtils := &linuxKernel.KernelUtils{}

	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}

	p := proxy.NewProxy(config.VIPs, proxySubnets, netUtils)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)

	clientConfig := &client.Config{
		Name:           config.Name,
		RequestTimeout: config.RequestTimeout,
	}
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, p, netUtils)
	client := getNSC(ctx, clientConfig, nsmAPIClient, p, interfaceMonitorClient)
	defer client.Close()
	go startNSC(client, config.NetworkServiceName)

	labels := map[string]string{}
	if config.Host != "" {
		labels["host"] = config.Host
	}
	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		MaxTokenLifetime: config.MaxTokenLifetime,
		Labels:           labels,
	}
	interfaceMonitorServer := interfacemonitor.NewServer(interfaceMonitor, p, netUtils)
	ep := startNSE(ctx, endpointConfig, nsmAPIClient, p, interfaceMonitorServer, config.NSPService)
	defer ep.Delete()

	configWatcher := make(chan *configuration.Config, 10)
	configurationWatcher := configuration.NewWatcher(config.ConfigMapName, config.Namespace, configWatcher)
	go configurationWatcher.Start()

	for {
		select {
		case config := <-configWatcher:
			p.SetVIPs(config.VIPs)
		case <-ctx.Done():
			return
		}
	}
}

func getProxySubnets(config Config) ([]string, error) {
	proxySubnets := []string{}
	for index, subnetPool := range config.SubnetPools {
		_, err := netlink.ParseAddr(subnetPool)
		if err != nil {
			return []string{}, errors.Wrap(err, "Error Parsing subnet pool")
		}
		ipamClient, err := ipam.NewIpamClient(config.IPAMService)
		if err != nil {
			return []string{}, errors.Wrap(err, "Error creating new ipam client")
		}
		proxySubnet, err := ipamClient.AllocateSubnet(subnetPool, config.SubnetPrefixLengths[index])
		if err != nil {
			return []string{}, errors.Wrap(err, "Error AllocateSubnet")
		}
		proxySubnets = append(proxySubnets, proxySubnet)
	}
	return proxySubnets, nil
}

func getNSC(ctx context.Context,
	config *client.Config,
	nsmAPIClient *nsm.APIClient,
	p *proxy.Proxy,
	interfaceMonitorClient networkservice.NetworkServiceClient) client.NetworkServiceClient {

	networkServiceClient := chain.NewNetworkServiceClient(
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		interfacename.NewClient("nsc", &interfacename.RandomGenerator{}),
		ipcontext.NewClient(p),
		interfaceMonitorClient,
		sendfd.NewClient(),
	)
	fullMeshClient := client.NewFullMeshNetworkServiceClient(config, nsmAPIClient.GRPCClient, networkServiceClient)
	return fullMeshClient
}

func startNSC(fullMeshClient client.NetworkServiceClient, networkServiceName string) {
	err := fullMeshClient.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: networkServiceName,
			Labels:         map[string]string{"forwarder": "forwarder-vpp"},
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
		logrus.Fatalf("fullMeshClient.Request err: %+v", err)
	}
}

func startNSE(ctx context.Context,
	config *endpoint.Config,
	nsmAPIClient *nsm.APIClient,
	p *proxy.Proxy,
	interfaceMonitorServer networkservice.NetworkServiceServer,
	nspService string) *endpoint.Endpoint {

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfacename.NewServer("nse", &interfacename.RandomGenerator{}),
		ipcontext.NewServer(p),
		interfaceMonitorServer,
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
	return ep
}
