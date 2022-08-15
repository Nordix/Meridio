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
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/ipam/conduitwatcher"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/nordix/meridio/pkg/ipam/trench"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
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

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	logger := log.NewLogrusLogger()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(log.DebugLevel)

	// create and start health server
	_ = health.CreateChecker(context.Background())
	var config Config
	err := envconfig.Process("ipam", &config)
	if err != nil {
		logger.Fatal("%v", err)
	}
	logger.Info("rootConf: %+v", config)

	logger.SetLevel(func() log.Level {
		l, err := log.ParseLevel(config.LogLevel)
		if err != nil {
			logger.Fatal("invalid log level %s", config.LogLevel)
		}
		return l
	}())

	prefixLengths, cidrs := GetPrefixLengthsAndCIDRs(&config)

	store, err := sqlite.New(config.Datasource)
	if err != nil {
		logger.Fatal("invalid log level %s", config.LogLevel)
	}

	trenches, conduitWatcherTrenches, err := SetupTrenches(ctx, store, config.TrenchName, prefixLengths, cidrs)
	if err != nil {
		logger.Fatal("error setup trenches %s", config.LogLevel)
	}

	conduitWatcherLogger := logger.WithField(log.SubSystem, "Conduit-Watcher")
	go func() {
		err := conduitwatcher.Start(ctx, config.NSPService, config.TrenchName, conduitWatcherTrenches, conduitWatcherLogger)
		if err != nil {
			conduitWatcherLogger.Fatal("error starting conduit watcher %s", config.LogLevel)
		}
	}()

	ipamServerLogger := logger.WithField(log.SubSystem, "IPAM-Server")
	StartServer(&config, trenches, prefixLengths, ipamServerLogger)
}

func GetPrefixLengthsAndCIDRs(config *Config) (map[ipamAPI.IPFamily]*types.PrefixLengths, map[ipamAPI.IPFamily]string) {
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
	return prefixLengths, cidrs
}

func StartServer(config *Config, trenches map[ipamAPI.IPFamily]types.Trench, prefixLengths map[ipamAPI.IPFamily]*types.PrefixLengths, logger log.Logger) {
	ipamServer, err := ipam.NewServer(trenches, prefixLengths, logger)
	if err != nil {
		logger.Fatal("unable to create ipam server: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(
		credentials.GetServer(context.Background()),
	))
	ipamAPI.RegisterIpamServer(server, ipamServer)
	healthServer := grpcHealth.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	logger.Info("start the service (port: %v)", config.Port)
	listener, err := net.Listen("tcp", fmt.Sprintf("[::]:%d", config.Port))
	if err != nil {
		logger.Fatal("failed to listen: %v", err)
	}

	if err := server.Serve(listener); err != nil {
		logger.Error("failed to serve: %v", err)
	}
}

func SetupTrenches(ctx context.Context,
	store types.Storage,
	trenchName string,
	prefixLengths map[ipamAPI.IPFamily]*types.PrefixLengths,
	cidrs map[ipamAPI.IPFamily]string) (map[ipamAPI.IPFamily]types.Trench, []conduitwatcher.Trench, error) {
	trenches := map[ipamAPI.IPFamily]types.Trench{}
	conduitWatcherTrenches := []conduitwatcher.Trench{}
	for ipFamily, cidr := range cidrs {
		name := ipam.GetTrenchName(trenchName, ipFamily)
		prefix := prefix.New(name, cidr, nil)
		newTrench, err := trench.New(context.TODO(), prefix, store, prefixLengths[ipFamily])
		if err != nil {
			return nil, nil, err
		}
		conduitWatcherTrenches = append(conduitWatcherTrenches, newTrench)
		trenches[ipFamily] = newTrench
	}
	return trenches, conduitWatcherTrenches, nil
}
