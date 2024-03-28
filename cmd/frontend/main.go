/*
Copyright (c) 2021-2024 Nordix Foundation

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

	"github.com/go-logr/logr"
	"github.com/kelseyhightower/envconfig"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	feConfig "github.com/nordix/meridio/cmd/frontend/internal/config"
	"github.com/nordix/meridio/cmd/frontend/internal/env"
	"github.com/nordix/meridio/cmd/frontend/internal/frontend"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/metrics"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/nordix/meridio/pkg/security/credentials"
)

func printHelp() {
	fmt.Println(`
frontend --
  The frontend process in https://github.com/Nordix/Meridio uses BGP (Bird)
  to attract traffic to Virtual IP (VIP) addresses.
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

	config := &env.Config{}
	if err := envconfig.Process("nfe", config); err != nil {
		panic(err)
	}
	logger := log.New("Meridio-frontend", config.LogLevel)
	logger.Info("Config read", "config", config)

	rootCtx, cancelRootCtx := context.WithCancel(logr.NewContext(context.Background(), logger))
	defer cancelRootCtx()

	ctx, cancel := signal.NotifyContext(
		rootCtx,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(logger, "Unable to get hostname", "error", err)
	}

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinessSubservices(ctx, health.FEReadinessServices...); err != nil {
		logger.Error(err, "RegisterReadinesSubservices")
	}

	// connect NSP
	logger.Info("Dial NSP", "NSPService", config.NSPService)
	conn, err := connectNSP(ctx, config)
	if err != nil {
		log.Fatal(logger, "Dial NSP", "error", err)
	}
	defer conn.Close()

	// monitor status of NSP connection and adjust probe status accordingly
	if err := connection.Monitor(ctx, health.NSPCliSvc, conn); err != nil {
		logger.Error(err, "NSP connection state monitor")
	}

	gatewayMetrics := frontend.NewGatewayMetrics([]metric.ObserveOption{
		metric.WithAttributes(attribute.String("trench", config.TrenchName)),
		metric.WithAttributes(attribute.String("attractor", config.AttractorName)),
	})

	// connect loadbalancer
	logger.Info("Dial loadbalancer", "LBService", config.LBSocket)
	lbConn, err := connectLoadbalancer(ctx, config)
	if err != nil {
		log.Fatal(logger, "Dial loadbalancer", "error", err)
	}
	defer lbConn.Close()

	// create and start frontend service
	c := &feConfig.Config{
		Config:   config,
		NSPConn:  conn,
		LBConn:   lbConn,
		Hostname: hostname,
	}
	fe := frontend.NewFrontEndService(ctx, c, gatewayMetrics)
	defer fe.CleanUp()

	if err := fe.Init(); err != nil {
		cancel()
		log.Fatal(logger, "Init failed", "error", err)
	}

	feErrCh := make(chan error, 1)
	defer close(feErrCh)
	fe.Start(rootCtx, feErrCh)
	defer func() {
		deleteCtx, deleteClose := context.WithTimeout(rootCtx, 3*time.Second)
		defer deleteClose()
		fe.Stop(deleteCtx)
	}()
	exitOnErrCh(logr.NewContext(ctx, logger.WithValues("func", "Start")), cancel, feErrCh)

	// monitor routing sessions
	monErrCh := make(chan error, 1)
	defer close(monErrCh)
	fe.Monitor(ctx, monErrCh)
	exitOnErrCh(logr.NewContext(ctx, logger.WithValues("func", "Monitor")), cancel, monErrCh)

	// start watching events of interest via NSP
	go watchConfig(ctx, cancel, c, fe)

	interfaceMetrics := linuxKernel.NewInterfaceMetrics([]metric.ObserveOption{
		metric.WithAttributes(attribute.String("trench", config.TrenchName)),
		metric.WithAttributes(attribute.String("attractor", config.AttractorName)),
	})
	interfaceMetrics.Register(config.ExternalInterface)

	if config.MetricsEnabled {
		func() {
			_, err = metrics.Init(ctx)
			if err != nil {
				logger.Error(err, "Unable to init metrics collector")
				cancel()
				return
			}

			err = interfaceMetrics.Collect()
			if err != nil {
				logger.Error(err, "Unable to start interface metrics collector")
				cancel()
				return
			}

			err = gatewayMetrics.Collect()
			if err != nil {
				logger.Error(err, "Unable to start gateway metrics collector")
				cancel()
				return
			}

			metricsServer := metrics.Server{
				IP:   "",
				Port: config.MetricsPort,
			}
			go func() {
				err := metricsServer.Start(ctx)
				if err != nil {
					logger.Error(err, "Unable to start metrics server")
					cancel()
				}
			}()
		}()
	}

	<-ctx.Done()
	logger.Info("FE shutting down")
}

func exitOnErrCh(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) {
	logger := log.FromContextOrGlobal(ctx).WithValues("func", "exitOnErrCh")
	// If we already have an error, log it and exit
	select {
	case err, ok := <-errCh:
		if ok {
			logger.Error(err, "already have an error")
			cancel()
		}
	default:
	}
	// Otherwise wait for an error in the background to log and cancel
	go func(ctx context.Context, errCh <-chan error) {
		select {
		case <-ctx.Done():
			logger.V(1).Info("context closed")
		case err, ok := <-errCh:
			if ok {
				logger.Error(err, "got an error")
				cancel()
			}
		}
	}(ctx, errCh)
}

func watchConfig(ctx context.Context, cancel context.CancelFunc, c *feConfig.Config, fe *frontend.FrontEndService) {
	logger := log.FromContextOrGlobal(ctx).WithValues("func", "watchConfig")
	if err := fe.WaitStart(ctx); err != nil {
		logger.Error(err, "WaitStart")
		cancel()
	}
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(c.NSPConn)
	attractorToWatch := &nspAPI.Attractor{
		Name: c.AttractorName,
		Trench: &nspAPI.Trench{
			Name: c.TrenchName,
		},
	}

	err := retry.Do(func() error {
		return watchAttractor(ctx, configurationManagerClient, attractorToWatch, fe)
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		logger.Error(err, "Attractor watcher")
		cancel()
	}
}

func watchAttractor(ctx context.Context, cli nspAPI.ConfigurationManagerClient, toWatch *nspAPI.Attractor, fe *frontend.FrontEndService) error {
	logger := log.FromContextOrGlobal(ctx)
	watchAttractor, err := cli.WatchAttractor(ctx, toWatch)
	if err != nil {
		return fmt.Errorf("failed to create attractor watcher: %w", err)
	}
	for {
		attractorResponse, err := watchAttractor.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Info("Attractor watcher closing down")
			return fmt.Errorf("attractor watcher recv error: %w", err)
		}
		logger.Info("Attractor config change event")
		if err := fe.SetNewConfig(ctx, attractorResponse.Attractors); err != nil {
			return fmt.Errorf("failed to set new Attractor config: %w", err)
		}
	}
	return nil
}

func connectNSP(ctx context.Context, config *env.Config) (*grpc.ClientConn, error) {
	grpcBackoffCfg := backoff.DefaultConfig
	if grpcBackoffCfg.MaxDelay != config.GRPCMaxBackoff {
		grpcBackoffCfg.MaxDelay = config.GRPCMaxBackoff
	}
	conn, err := grpc.DialContext(ctx,
		config.NSPService,
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
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
		err = fmt.Errorf("failed to dial NSP: %w", err)
	}

	return conn, err
}

func connectLoadbalancer(ctx context.Context, config *env.Config) (*grpc.ClientConn, error) {
	lbConn, err := grpc.DialContext(ctx,
		func() string {
			switch config.LBSocket.Scheme {
			case "unix":
				return config.LBSocket.String()
			case "tcp":
				return config.LBSocket.Host
			}
			return config.LBSocket.String()
		}(),
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: func() backoff.Config {
				grpcBackoffCfg := backoff.DefaultConfig
				if grpcBackoffCfg.MaxDelay != config.GRPCMaxBackoff {
					grpcBackoffCfg.MaxDelay = config.GRPCMaxBackoff
				}
				return grpcBackoffCfg
			}(),
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: config.GRPCKeepaliveTime,
		}),
	)
	if err != nil {
		err = fmt.Errorf("failed to dial loadbalancer: %w", err)
	}

	return lbConn, err
}
