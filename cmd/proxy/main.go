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
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	sriovtoken "github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/configuration"
	endpointOld "github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/probe"
	"github.com/nordix/meridio/pkg/ipam"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/endpoint"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/nordix/meridio/pkg/nsm/ipcontext"
	"github.com/nordix/meridio/pkg/nsm/service"
	"github.com/nordix/meridio/pkg/proxy"
	proxyHealth "github.com/nordix/meridio/pkg/proxy/health"
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
	logrus.Infof("rootConf: %+v", config)

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.ProxyReadinessServices...); err != nil {
		logrus.Warnf("%v", err)
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

	p := proxy.NewProxy(proxySubnets, netUtils)

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
		ConnectTo:      config.ConnectTo,
	}
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, p, netUtils)
	client := getNSC(ctx, clientConfig, nsmAPIClient, p, interfaceMonitorClient)
	defer client.Close()
	go startNSC(client, config.NetworkServiceName)

	labels := map[string]string{}
	if config.Host != "" {
		labels["nodeName"] = config.Host
	}
	endpointConfig := &endpointOld.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		MaxTokenLifetime: config.MaxTokenLifetime,
		Labels:           labels,
	}
	interfaceMonitorServer := interfacemonitor.NewServer(interfaceMonitor, p, netUtils)
	ep := startNSE(ctx, endpointConfig, nsmAPIClient, p, interfaceMonitorServer)
	probe.CreateAndRunGRPCHealthProbe(ctx, health.NSMEndpointSvc, probe.WithAddress(ep.Server.GetUrl()), probe.WithSpiffe())
	defer func() {
		err := ep.Delete(context.Background())
		if err != nil {
			logrus.Errorf("Err delete NSE: %v", err)
		}
	}()

	// TODO: use NSP based config watcher
	configWatcher := make(chan *configuration.OperatorConfig)
	configurationWatcher := configuration.NewOperatorWatcher(config.ConfigMapName, config.Namespace, configWatcher)
	go configurationWatcher.Start()
	health.SetServingStatus(ctx, health.NSPCliSvc, true)

	for {
		select {
		case config := <-configWatcher:
			p.SetVIPs(configuration.AddrListFromVipConfig(config.VIPs))
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
		updatepath.NewClient(config.Name),
		serialize.NewClient(),
		refresh.NewClient(ctx),
		metadata.NewClient(),
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		interfacename.NewClient("nsc", &interfacename.RandomGenerator{}),
		ipcontext.NewClient(p),
		interfaceMonitorClient,
		proxyHealth.NewClient(),
		authorize.NewClient(),
		sendfd.NewClient(),
	)
	fullMeshClient := client.NewFullMeshNetworkServiceClient(ctx, config, nsmAPIClient, networkServiceClient)
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
	config *endpointOld.Config,
	nsmAPIClient *nsm.APIClient,
	p *proxy.Proxy,
	interfaceMonitorServer networkservice.NetworkServiceServer) *endpoint.Endpoint {

	logrus.Infof("startNSE")
	additionalFunctionality := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfacename.NewServer("nse", &interfacename.RandomGenerator{}),
		ipcontext.NewServer(p),
		interfaceMonitorServer,
		sendfd.NewServer(),
	}

	ns := &registry.NetworkService{
		Name:    config.ServiceName,
		Payload: payload.Ethernet,
		Matches: []*registry.Match{
			{
				SourceSelector: make(map[string]string),
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"nodeName": "{{.nodeName}}",
						},
					},
				},
			},
		},
	}

	service := service.New(nsmAPIClient.NetworkServiceRegistryClient, ns)
	err := service.Register(ctx)
	if err != nil {
		logrus.Errorf("Err creating NSE: %v", err)
	}

	nse := &registry.NetworkServiceEndpoint{
		Name:                config.Name,
		NetworkServiceNames: []string{config.ServiceName},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			config.ServiceName: {
				Labels: config.Labels,
			},
		},
	}

	endpoint, err := endpoint.New(config.MaxTokenLifetime,
		nsmAPIClient.NetworkServiceEndpointRegistryClient,
		nse,
		additionalFunctionality...)
	if err != nil {
		logrus.Errorf("Err creating NSE: %v", err)
	}
	err = endpoint.Register(ctx)
	if err != nil {
		logrus.Errorf("Err registring NSE: %v", err)
	}
	return endpoint
}
