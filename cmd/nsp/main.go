/*
Copyright (c) 2021-2023 Nordix Foundation

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
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/configuration/manager"
	"github.com/nordix/meridio/pkg/configuration/monitor"
	"github.com/nordix/meridio/pkg/configuration/registry"
	"github.com/nordix/meridio/pkg/debug"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/probe"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/nsp"

	nsmlog "github.com/networkservicemesh/sdk/pkg/tools/log"
	keepAliveRegistry "github.com/nordix/meridio/pkg/nsp/registry/keepalive"
	sqliteRegistry "github.com/nordix/meridio/pkg/nsp/registry/sqlite"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	grpcHealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func printHelp() {
	fmt.Println(`
nsp --
  The nsp (Network Service Platform) process in https://github.com/Nordix/Meridio
  handles notifications on (un)registration.
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

	var config Config
	err := envconfig.Process("nsp", &config)
	if err != nil {
		panic(err)
	}
	logger := log.New("Meridio-nsp", config.LogLevel)
	logger.Info("Configuration read", "config", config)

	ctx, cancel := signal.NotifyContext(
		logr.NewContext(context.Background(), logger),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	if config.LogLevel == "TRACE" {
		nsmlog.EnableTracing(true)
		// Work-around for hard-coded logrus dependency in NSM
		logrus.SetLevel(logrus.TraceLevel)
	}
	logger.Info("NSM trace", "enabled", nsmlog.IsTracingEnabled())
	ctx = nsmlog.WithLog(ctx, log.NSMLogger(logger)) // allow NSM logs

	// Set up dynamic log level change via signals
	log.SetupLevelChangeOnSignal(ctx, map[os.Signal]string{
		syscall.SIGUSR1: config.LogLevel,
		syscall.SIGUSR2: "TRACE",
	}, log.WithNSMLogger())

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinessSubservices(ctx); err != nil {
		logger.Error(err, "RegisterReadinessSubservices")
	}
	if err := health.RegisterLivenessSubservices(ctx, health.NSPLivenessServices...); err != nil {
		logger.Error(err, "RegisterLivenessSubservices")
	}

	// configuration
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationMonitor, err := monitor.New(config.ConfigMapName, config.Namespace, configurationRegistry)
	if err != nil {
		log.Fatal(logger, "Unable to start configuration monitor", "error", err)
	}
	go configurationMonitor.Start(context.Background())
	watcherNotifier := manager.NewWatcherNotifier(configurationRegistry, configurationEventChan)
	go watcherNotifier.Start(context.Background())
	configurationManagerServer := manager.NewServer(watcherNotifier)

	// target registry
	sqlr, err := sqliteRegistry.New(config.Datasource)
	if err != nil {
		log.Fatal(logger, "Unable create sqlite registry", "error", err)
	}
	keepAliveRegistry, err := keepAliveRegistry.New(
		keepAliveRegistry.WithRegistry(sqlr),
		keepAliveRegistry.WithTimeout(config.EntryTimeout),
	)
	if err != nil {
		log.Fatal(logger, "Unable create keepalive registry", "error", err)
	}
	targetRegistryServer := nsp.NewServer(keepAliveRegistry)

	server := grpc.NewServer(grpc.Creds(
		credentials.GetServer(context.Background()),
	))
	nspAPI.RegisterTargetRegistryServer(server, targetRegistryServer)
	nspAPI.RegisterConfigurationManagerServer(server, configurationManagerServer)
	healthServer := grpcHealth.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	listener, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", config.Port))
	logger.Info("Start the service", "service-port", config.Port)
	if err != nil {
		log.Fatal(logger, "NSP Service: failed to listen", "error", err)
	}

	// internal probe checking health of IPAM server
	probe.CreateAndRunGRPCHealthProbe(
		ctx,
		health.NSPSvc,
		probe.WithAddress(fmt.Sprintf(":%s", config.Port)),
		probe.WithSpiffe(),
		probe.WithRPCTimeout(config.GRPCProbeRPCTimeout),
	)

	if err := startServer(ctx, server, listener); err != nil {
		logger.Error(err, "NSP Service: failed to serve")
	}
}

func startServer(ctx context.Context, server *grpc.Server, listener net.Listener) error {
	defer func() {
		_ = listener.Close()
	}()
	// montior context in separate goroutine to be able to stop server
	go func() {
		<-ctx.Done()
		server.Stop()
	}()

	err := server.Serve(listener)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
