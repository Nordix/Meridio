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
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
	"github.com/nordix/meridio/pkg/health/probe"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
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

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.IPAMReadinessServices...); err != nil {
		logrus.Warnf("%v", err)
	}

	var config Config
	err := envconfig.Process("ipam", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	logrus.Infof("rootConf: %+v", config)

	logrus.SetLevel(func() logrus.Level {

		l, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			logrus.Fatalf("invalid log level %s", config.LogLevel)
		}
		return l
	}())

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
		logrus.Fatalf("Dial NSP err: %v", err)
	}
	defer conn.Close()

	// monitor status of NSP connection and adjust probe status accordingly
	if err := connection.Monitor(ctx, health.NSPCliSvc, conn); err != nil {
		logrus.Warnf("NSP connection state monitor err: %v", err)
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
	ipamServer, err := ipam.NewServer(config.Datasource, config.TrenchName, conn, cidrs, prefixLengths)
	if err != nil {
		logrus.Fatalf("Unable to create ipam server: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(
		credentials.GetServer(ctx),
	))
	ipamAPI.RegisterIpamServer(server, ipamServer)
	healthServer := grpcHealth.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	logrus.Infof("IPAM Service: Start the service (port: %v)", config.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("[::]:%d", config.Port))
	if err != nil {
		logrus.Fatalf("IPAM Service: failed to listen: %v", err)
	}

	// internal probe checking health of IPAM server
	probe.CreateAndRunGRPCHealthProbe(ctx, health.IPAMSvc, probe.WithAddress(fmt.Sprintf(":%d", config.Port)), probe.WithSpiffe())

	if err := startServer(ctx, server, listener); err != nil {
		logrus.Errorf("IPAM Service: failed to serve: %v", err)
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
