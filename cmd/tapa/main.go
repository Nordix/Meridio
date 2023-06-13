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
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
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
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	nsmlog "github.com/networkservicemesh/sdk/pkg/tools/log"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap"
	"github.com/nordix/meridio/pkg/health"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	grpcHealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func printHelp() {
	fmt.Println(`
tapa --
  The tapa process in https://github.com/Nordix/Meridio
  serves as ambassador in target PODs.
  This program shall be started as a Kubernetes sidecar container.`)
}

var version = "(unknown)"

func main() {
	ver := flag.Bool("version", false, "Print version and quit")
	help := flag.Bool("help", false, "Print help and quit")
	flag.Parse()
	if *ver {
		fmt.Println(version)
		os.Exit(0)
	}
	if *help {
		printHelp()
		os.Exit(0)
	}

	var config Config
	err := envconfig.Process("meridio", &config)
	if err != nil {
		panic(err)
	}

	logger := log.New("Meridio-tapa", config.LogLevel)
	logger.Info("Configuration read", "config", config)
	if err := config.IsValid(); err != nil {
		log.Fatal(logger, "config.IsValid", "error", err)
	}

	if config.LogLevel == "TRACE" {
		nsmlog.EnableTracing(true) // enable tracing in NSM
		logrus.SetLevel(logrus.TraceLevel)
	}

	logger.Info("NSM trace", "enabled", nsmlog.IsTracingEnabled())
	nsmlogger := log.NSMLogger(logger)
	nsmlog.SetGlobalLogger(nsmlogger)

	// Create context with loggers
	sigCtx, cancelSignalCtx := signal.NotifyContext(
		logr.NewContext(context.Background(), logger),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancelSignalCtx()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = nsmlog.WithLog(ctx, nsmlogger)

	netUtils := &linuxKernel.KernelUtils{}

	// create and start health server
	sigCtx = health.CreateChecker(sigCtx)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.NSMSocket,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.Timeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)
	defer nsmAPIClient.Delete()

	additionalFunctionality := []networkservice.NetworkServiceClient{
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		interfacename.NewClient("nsm-", &interfacename.CounterGenerator{}),
		sendfd.NewClient(),
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

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(&nsmAPIClient.Config.ConnectTo), nsmAPIClient.GRPCDialOption...)
	if err != nil {
		log.Fatal(logger, "dial to NSMgr", "error", err)
	}
	defer cc.Close()
	monitorClient := networkservice.NewMonitorConnectionClient(cc)

	if err := os.RemoveAll(config.Socket); err != nil {
		log.Fatal(logger, "removing socket", "error", err)
	}
	lis, err := net.Listen("unix", config.Socket)
	if err != nil {
		log.Fatal(logger, "listen on unix socket", "error", err)
	}
	if err := os.Chmod(config.Socket, os.ModePerm); err != nil {
		logger.Error(err, "changing unix socket permission")
	}
	s := grpc.NewServer()
	defer s.Stop()

	ambassador, err := tap.New(
		config.Name,
		config.Namespace,
		config.Node,
		networkServiceClient,
		monitorClient,
		config.NSPServiceName,
		config.NSPServicePort,
		config.NSPEntryTimeout,
		config.GRPCMaxBackoff,
		netUtils,
	)
	if err != nil {
		log.Fatal(logger, "creating new tap ambassador", "error", err)
	}
	defer func() {
		err = ambassador.Delete(context.TODO())
		if err != nil {
			logger.Error(err, "deleting ambassador")
		}
	}()

	healthServer := grpcHealth.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	ambassadorAPI.RegisterTapServer(s, ambassador)

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Error(err, "Ambassador: failed to serve")
		}
	}()

	<-sigCtx.Done()
}
