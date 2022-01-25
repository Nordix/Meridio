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
	"io"
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
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/client"
	endpointOld "github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/probe"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/endpoint"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/ipcontext"
	"github.com/nordix/meridio/pkg/nsm/service"
	"github.com/nordix/meridio/pkg/nsp"

	"github.com/nordix/meridio/pkg/proxy"
	proxyHealth "github.com/nordix/meridio/pkg/proxy/health"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

	logrus.SetLevel(func() logrus.Level {

		l, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			logrus.Fatalf("invalid log level %s", config.LogLevel)
		}
		if l == logrus.TraceLevel {
			log.EnableTracing(true) // enable tracing in NSM
		}
		return l
	}())
	ctx = log.WithLog(ctx, logruslogger.New(ctx)) // allow NSM logs

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.ProxyReadinessServices...); err != nil {
		logrus.Warnf("%v", err)
	}

	netUtils := &linuxKernel.KernelUtils{}

	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}

	conn, err := grpc.Dial(config.IPAMService,
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Fatalf("Error dialing IPAM: %+v", err)
	}
	ipamClient := ipamAPI.NewIpamClient(conn)
	conduit := &nspAPI.Conduit{
		Name: config.Conduit,
		Trench: &nspAPI.Trench{
			Name: config.Trench,
		},
	}
	p := proxy.NewProxy(conduit, config.Host, ipamClient, config.IPFamily, netUtils)

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

	configurationContext, configurationCancel := context.WithCancel(ctx)
	defer configurationCancel()
	logrus.Debugf("Dial NSP (%v)", nsp.GetService(config.NSPServiceName, config.Trench, config.Namespace, config.NSPServicePort))
	nspConn, err := grpc.Dial(nsp.GetService(config.NSPServiceName, config.Trench, config.Namespace, config.NSPServicePort),
		grpc.WithTransportCredentials(
			credentials.GetClient(configurationContext),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Fatalf("Dial err: %v", err)
	}

	configurationManagerClient := nspAPI.NewConfigurationManagerClient(nspConn)
	vipWatcher, err := configurationManagerClient.WatchVip(configurationContext, &nspAPI.Vip{
		Trench: &nspAPI.Trench{
			Name: config.Trench,
		},
	})
	if err != nil {
		logrus.Fatalf("WatchVip err: %v", err)
	}
	logrus.Debugf("Connected to NSP")
	health.SetServingStatus(ctx, health.NSPCliSvc, true)
	for {
		vipResponse, err := vipWatcher.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Warnf("err vipWatcher.Recv: %v", err) // todo
			break
		}
		p.SetVIPs(vipResponse.ToSlice())
	}
	<-ctx.Done()
}

func getNSC(ctx context.Context,
	config *client.Config,
	nsmAPIClient *nsm.APIClient,
	p *proxy.Proxy,
	interfaceMonitorClient networkservice.NetworkServiceClient) client.NetworkServiceClient {

	log.FromContext(ctx).Infof("Get New NSC")
	// Note: naming the interface is left to NSM (refer to getNameFromConnection())
	// However NSM does not seem to ensure uniqueness either. Might need to revisit...
	additionalFunctionality := chain.NewNetworkServiceClient(
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		ipcontext.NewClient(p),
		interfaceMonitorClient,
		proxyHealth.NewClient(),
	)
	fullMeshClient := client.NewFullMeshNetworkServiceClient(ctx, config, nsmAPIClient, additionalFunctionality)
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
		// Note: naming the interface is left to NSM (refer to getNameFromConnection())
		// However NSM does not seem to ensure uniqueness either. Might need to revisit...
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		ipcontext.NewServer(p),
		interfaceMonitorServer,
		sendfd.NewServer(),
	}

	ns := &registry.NetworkService{
		Name:    config.ServiceName,
		Payload: payload.Ethernet,
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{
					"nodeName": "{{.nodeName}}",
				},
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
	logrus.Debugf("Create NS: %v", ns)

	service := service.New(nsmAPIClient.NetworkServiceRegistryClient, ns)
	err := service.Register(ctx)
	if err != nil {
		logrus.Errorf("Err registering NS: %v", err)
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
	logrus.Debugf("Create NSE: %v", nse)

	endpoint, err := endpoint.New(config.MaxTokenLifetime,
		nsmAPIClient.NetworkServiceEndpointRegistryClient,
		nse,
		additionalFunctionality...)
	if err != nil {
		logrus.Errorf("Err creating NSE: %v", err)
	}
	err = endpoint.Register(ctx)
	if err != nil {
		logrus.Errorf("Err registering NSE: %v", err)
	}
	return endpoint
}
