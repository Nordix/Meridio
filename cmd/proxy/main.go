/*
Copyright (c) 2021-2023 Nordix Foundation
Copyright (c) 2024-2025 OpenInfra Foundation Europe

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
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	nsmlog "github.com/networkservicemesh/sdk/pkg/tools/log"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/cmd/proxy/internal/config"
	"github.com/nordix/meridio/cmd/proxy/internal/service"
	"github.com/nordix/meridio/pkg/debug"
	endpointOld "github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
	"github.com/nordix/meridio/pkg/health/probe"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	nsmmonitor "github.com/nordix/meridio/pkg/nsm/monitor"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/nordix/meridio/pkg/retry"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/proxy"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

func printHelp() {
	fmt.Println(`
proxy --
  The proxy process in https://github.com/Nordix/Meridio
  acts as a bridge between load-balancers and targets.
  This program shall be started in a Kubernetes container.`)
}

var version = "(unknown)"

func main() {
	ver := flag.Bool("version", false, "Print version and quit")
	debugCmd := flag.Bool("debug", false, "Print the debug information and quit")
	help := flag.Bool("help", false, "Print help and quit")
	flag.Parse()
	if *ver {
		fmt.Println(version)
		os.Exit(0)
	}
	if *debugCmd {
		debug.MeridioVersion = version
		fmt.Println(debug.Collect().String())
		os.Exit(0)
	}
	if *help {
		printHelp()
		os.Exit(0)
	}

	var config config.Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		panic(err)
	}

	logger := log.New("Meridio-proxy", config.LogLevel)
	logger.Info("Configuration read", "config", config)
	if err := config.IsValid(); err != nil {
		log.Fatal(logger, "config.IsValid", "error", err)
	}

	ctx, cancel := context.WithCancel(
		logr.NewContext(context.Background(), logger))
	defer cancel()

	// allow NSM logs
	if config.LogLevel == "TRACE" {
		nsmlog.EnableTracing(true)
		// Work-around for hard-coded logrus dependency in NSM
		logrus.SetLevel(logrus.TraceLevel)
	}
	logger.Info("NSM trace", "enabled", nsmlog.IsTracingEnabled())
	nsmlogger := log.NSMLogger(logger)
	nsmlog.SetGlobalLogger(nsmlogger)
	ctx = nsmlog.WithLog(ctx, nsmlogger)

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinessSubservices(ctx, health.ProxyReadinessServices...); err != nil {
		logger.Error(err, "RegisterReadinessSubservices")
	}
	if err := health.RegisterLivenessSubservices(ctx, health.ProxyLivenessServices...); err != nil {
		logger.Error(err, "RegisterLivenessSubservices")
	}

	// context enabling graceful termiantion on signals
	signalCtx, cancelSignalCtx := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancelSignalCtx()

	netUtils := &linuxKernel.KernelUtils{}
	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		log.Fatal(logger, "Creating link monitor", "error", err)
	}

	// connect IPAM the proxy relies on to assign IPs both locally and remote via nsc and nse
	logger.Info("Dial IPAM", "service", config.IPAMService)
	grpcBackoffCfg := backoff.DefaultConfig
	if grpcBackoffCfg.MaxDelay != config.GRPCMaxBackoff {
		grpcBackoffCfg.MaxDelay = config.GRPCMaxBackoff
	}
	conn, err := grpc.DialContext(signalCtx,
		config.IPAMService,
		grpc.WithTransportCredentials(
			credentials.GetClient(ctx),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: grpcBackoffCfg,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: config.GRPCKeepaliveTime,
		}),
	)
	if err != nil {
		log.Fatal(logger, "Dialing IPAM", "error", err)
	}
	defer conn.Close()

	// monitor status of IPAM connection and adjust probe status accordingly
	if err := connection.Monitor(signalCtx, health.IPAMCliSvc, conn); err != nil {
		logger.Error(err, "IPAM connection state monitor")
		cancelSignalCtx()
		return
	}

	// connect NSP
	logger.Info("Dial NSP", "service", nsp.GetService(config.NSPServiceName, config.Trench, config.Namespace, config.NSPServicePort))
	nspConn, err := grpc.DialContext(signalCtx,
		nsp.GetService(config.NSPServiceName, config.Trench, config.Namespace, config.NSPServicePort),
		grpc.WithTransportCredentials(
			credentials.GetClient(ctx),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: grpcBackoffCfg,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: config.GRPCKeepaliveTime,
		}),
	)
	if err != nil {
		log.Fatal(logger, "Dialing NSP", "error", err)
	}
	defer nspConn.Close()

	// monitor status of NSP connection and adjust probe status accordingly
	if err := connection.Monitor(signalCtx, health.NSPCliSvc, nspConn); err != nil {
		logger.Error(err, "NSP connection state monitor")
		cancelSignalCtx()
		return
	}

	ipamClient := ipamAPI.NewIpamClient(conn)
	conduit := &nspAPI.Conduit{
		Name: config.Conduit,
		Trench: &nspAPI.Trench{
			Name: config.Trench,
		},
	}
	p, err := proxy.NewProxy(signalCtx, conduit, config.Host, ipamClient, config.IPFamily, netUtils)
	if err != nil {
		logger.Error(err, "Proxy create")
		cancelSignalCtx()
		return
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(ctx, 8*time.Second)
		p.Close(closeCtx)
		closeCancel()
	}()

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
		GRPCMaxBackoff:   config.GRPCMaxBackoff,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)
	defer nsmAPIClient.Delete()

	// connect NSMgr and start NSM connection monitoring (used to log events of interest at the moment)
	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(&nsmAPIClient.Config.ConnectTo), nsmAPIClient.GRPCDialOption...)
	if err != nil {
		logger.Error(err, "Dialing NSMgr")
		cancelSignalCtx()
		return
	}
	defer cc.Close()
	monitorClient := networkservice.NewMonitorConnectionClient(cc)
	go nsmmonitor.ConnectionMonitor(ctx, config.Name, monitorClient)

	// create and start NSC that connects all remote NSE belonging to the right service
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, p, netUtils)
	nsmClient := service.GetNSC(ctx, &config, nsmAPIClient, p, interfaceMonitorClient, monitorClient)
	defer nsmClient.Close()
	go func() {
		service.StartNSC(nsmClient, config.NetworkServiceName)
		cancelSignalCtx() // let others with proper clean-up gracefully terminate
	}()

	// create and start NSE accepting ambassadors (targets) to connect
	labels := map[string]string{}
	if config.Host != "" {
		labels["nodeName"] = config.Host
	}
	endpointConfig := &endpointOld.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		MaxTokenLifetime: config.MaxTokenLifetime,
		Labels:           labels,
		MTU:              config.MTU,
		IPReleaseDelay:   config.IPReleaseDelay,
	}
	interfaceMonitorServer := interfacemonitor.NewServer(interfaceMonitor, p, netUtils)
	ep, err := service.StartNSE(ctx, endpointConfig, nsmAPIClient, p, interfaceMonitorServer)
	if err != nil {
		logger.Error(err, "Creating NSE")
		cancelSignalCtx()
		return
	}
	defer func() {
		deleteCtx, deleteClose := context.WithTimeout(ctx, 3*time.Second)
		defer deleteClose()
		if err := ep.Delete(deleteCtx); err != nil {
			logger.Error(err, "Delete NSE")
			return
		}
		log.Logger.V(1).Info("Deleted NSE")
	}()
	// internal probe checking health of NSE
	probe.CreateAndRunGRPCHealthProbe(
		signalCtx,
		health.NSMEndpointSvc,
		probe.WithAddress(ep.Server.GetUrl()),
		probe.WithSpiffe(),
		probe.WithRPCTimeout(config.GRPCProbeRPCTimeout),
	)

	// Start watching config events of interest
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(nspConn)
	logger.V(1).Info("Watch configuration")
	err = retry.Do(func() error {
		vipWatcher, err := configurationManagerClient.WatchVip(signalCtx, &nspAPI.Vip{
			Trench: &nspAPI.Trench{
				Name: config.Trench,
			},
		})
		if err != nil {
			return fmt.Errorf("configuration manager vip watcher client create failed: %w", err)
		}
		for {
			vipResponse, err := vipWatcher.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("configuration manager vip watcher client receive error: %w", err)
			}
			p.SetVIPs(vipResponse.ToSlice())
		}
		return nil
	}, retry.WithContext(signalCtx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil && status.Code(err) != codes.Canceled {
		logger.Error(err, "VIP watcher error")
	}

	logger.Info("Proxy shutting done")
}
