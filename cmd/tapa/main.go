/*
Copyright (c) 2021-2022 Nordix Foundation

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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	sriovtoken "github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	tapAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap"
	"github.com/nordix/meridio/pkg/health"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	grpcHealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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

	var config Config
	err := envconfig.Process("meridio", &config)
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

	logrus.SetLevel(func() logrus.Level {
		l, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			logrus.Fatalf("invalid log level %s", config.LogLevel)
		}
		return l
	}())

	netUtils := &linuxKernel.KernelUtils{}

	healthChecker, err := health.NewChecker(health.WithCtx(ctx))
	if err != nil {
		logrus.Fatalf("Unable to create Health checker: %v", err)
	}
	go func() {
		err := healthChecker.Start()
		if err != nil {
			logrus.Fatalf("Unable to start Health checker: %v", err)
		}
	}()

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.NSMSocket,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.Timeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)

	additionalFunctionality := []networkservice.NetworkServiceClient{
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient(kernel.WithInterfaceName("nsc"))),
		}),
		sendfd.NewClient(),
		dnscontext.NewClient(dnscontext.WithChainContext(ctx)),
		// excludedprefixes.NewClient(),
	}

	networkServiceClient := client.NewClient(ctx,
		client.WithClientURL(&nsmAPIClient.Config.ConnectTo),
		client.WithName(config.Name),
		client.WithAuthorizeClient(authorize.NewClient()),
		client.WithHealClient(heal.NewClient(ctx)),
		client.WithAdditionalFunctionality(additionalFunctionality...),
		client.WithDialTimeout(nsmAPIClient.Config.DialTimeout),
		client.WithDialOptions(nsmAPIClient.GRPCDialOption...),
	)

	if err := os.RemoveAll(config.Socket); err != nil {
		logrus.Fatalf("error removing socket: %v", err)
	}
	lis, err := net.Listen("unix", config.Socket)
	if err != nil {
		logrus.Fatalf("error listening on unix socket: %v", err)
	}
	s := grpc.NewServer()
	defer s.Stop()

	ambassador, err := tap.New(config.Name, config.Namespace, config.Node, networkServiceClient, config.NSPServiceName, config.NSPServicePort, netUtils)
	if err != nil {
		logrus.Fatalf("error creating new tap ambassador: %v", err)
	}
	defer func() {
		err = ambassador.Delete(context.TODO())
		if err != nil {
			logrus.Fatalf("Error deleting ambassador: %v", err)
		}
	}()

	healthServer := grpcHealth.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	tapAPI.RegisterTapServer(s, ambassador)

	go func() {
		if err := s.Serve(lis); err != nil {
			logrus.Errorf("TAP Ambassador: failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
}
