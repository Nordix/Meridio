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
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/kelseyhightower/envconfig"
	nsmlog "github.com/networkservicemesh/sdk/pkg/tools/log"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
	"github.com/nordix/meridio/pkg/health/probe"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/profiling"
	"github.com/nordix/meridio/pkg/security/credentials"
	"google.golang.org/grpc"
	grpcHealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func printHelp() {
	fmt.Println(`
ipam --
  The ipam process in https://github.com/Nordix/Meridio
  handles IP Address Management.
  This program shall be started in a Kubernetes container.`)
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
	err := envconfig.Process("ipam", &config)
	if err != nil {
		panic(err)
	}
	logger := log.New("Meridio-ipam", config.LogLevel)
	logger.Info("Configuration read", "config", config)

	ctx, cancel := signal.NotifyContext(
		logr.NewContext(context.Background(), logger),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	if config.ProfilingEnabled {
		go func() {
			mux := http.NewServeMux()
			profiling.AddProfilerHandlers(mux)
			err := http.ListenAndServe(fmt.Sprintf(":%d", config.ProfilingPort), mux)
			if err != nil {
				logger.Error(err, "err starting profiling")
			}
		}()
	}

	if config.LogLevel == "TRACE" {
		nsmlog.EnableTracing(true)
	}
	logger.Info("NSM trace", "enabled", nsmlog.IsTracingEnabled())
	ctx = nsmlog.WithLog(ctx, log.NSMLogger(logger)) // allow NSM logs

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.IPAMReadinessServices...); err != nil {
		logger.Error(err, "RegisterReadinesSubservices")
	}

	// connect NSP
	conn, err := grpc.DialContext(
		ctx,
		config.NSPService,
		grpc.WithTransportCredentials(
			credentials.GetClient(ctx),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		log.Fatal(logger, "Dial NSP err", "error", err)
	}
	defer conn.Close()

	// monitor status of NSP connection and adjust probe status accordingly
	if err := connection.Monitor(ctx, health.NSPCliSvc, conn); err != nil {
		logger.Error(err, "NSP connection state monitor")
	}

	prefixLengths := make(map[ipamAPI.IPFamily]*types.PrefixLengths)
	cidrs := make(map[ipamAPI.IPFamily]string)
	if strings.ToLower(config.IPFamily) == "ipv4" {
		prefixLengths[ipamAPI.IPFamily_IPV4] = types.NewPrefixLengths(config.ConduitPrefixLengthIPv4, config.NodePrefixLengthIPv4, 32)
		cidrs[ipamAPI.IPFamily_IPV4] = config.PrefixIPv4
	} else if strings.ToLower(config.IPFamily) == "ipv6" {
		prefixLengths[ipamAPI.IPFamily_IPV6] = types.NewPrefixLengths(config.ConduitPrefixLengthIPv6, config.NodePrefixLengthIPv6, 128)
		cidrs[ipamAPI.IPFamily_IPV6] = config.PrefixIPv6
	} else {
		prefixLengths[ipamAPI.IPFamily_IPV4] = types.NewPrefixLengths(config.ConduitPrefixLengthIPv4, config.NodePrefixLengthIPv4, 32)
		prefixLengths[ipamAPI.IPFamily_IPV6] = types.NewPrefixLengths(config.ConduitPrefixLengthIPv6, config.NodePrefixLengthIPv6, 128)
		cidrs[ipamAPI.IPFamily_IPV4] = config.PrefixIPv4
		cidrs[ipamAPI.IPFamily_IPV6] = config.PrefixIPv6
	}

	// cteate IPAM server
	ipamServer, err := ipam.NewServer(
		ctx, config.Datasource, config.TrenchName, conn, cidrs, prefixLengths)
	if err != nil {
		logger.Error(err, "Unable to create ipam server")
	}

	server := grpc.NewServer(grpc.Creds(
		credentials.GetServer(ctx),
	))
	ipamAPI.RegisterIpamServer(server, ipamServer)
	healthServer := grpcHealth.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	logger.Info("Start the service", "port", config.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("[::]:%d", config.Port))
	if err != nil {
		log.Fatal(logger, "Failed to listen", "error", err)
	}

	// internal probe checking health of IPAM server
	probe.CreateAndRunGRPCHealthProbe(ctx, health.IPAMSvc, probe.WithAddress(fmt.Sprintf(":%d", config.Port)), probe.WithSpiffe())

	if err := startServer(ctx, server, listener); err != nil {
		logger.Error(err, "IPAM Service: failed to serve: %v")
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
	return server.Serve(listener)
}
