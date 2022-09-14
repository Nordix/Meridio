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
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	feConfig "github.com/nordix/meridio/cmd/frontend/internal/config"
	"github.com/nordix/meridio/cmd/frontend/internal/env"
	"github.com/nordix/meridio/cmd/frontend/internal/frontend"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
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

	rootCtx, cancelRootCtx := context.WithCancel(context.Background())
	defer cancelRootCtx()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&nested.Formatter{})

	config := &env.Config{}
	if err := envconfig.Usage("nfe", config); err != nil {
		logrus.Fatal(err)
	}
	if err := envconfig.Process("nfe", config); err != nil {
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

	ctx, cancel := signal.NotifyContext(
		rootCtx,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.FEReadinessServices...); err != nil {
		logrus.Infof("%v", err)
	}

	// connect NSP
	logrus.Infof("Dial NSP (%v)", config.NSPService)
	conn, err := grpc.DialContext(ctx,
		config.NSPService,
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
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
		logrus.Infof("NSP connection state monitor err: %v", err)
	}

	// create and start frontend service
	c := &feConfig.Config{
		Config:  config,
		NSPConn: conn,
	}
	fe := frontend.NewFrontEndService(ctx, c)
	defer fe.CleanUp()
	health.SetServingStatus(ctx, health.TargetRegistryCliSvc, true) // NewFrontEndService() creates Target Registry Client

	if err := fe.Init(); err != nil {
		cancel()
		logrus.Fatalf("Init failed: %v", err)
	}

	feErrCh := make(chan error, 1)
	defer close(feErrCh)
	fe.Start(rootCtx, feErrCh)
	defer func() {
		deleteCtx, deleteClose := context.WithTimeout(rootCtx, 3*time.Second)
		defer deleteClose()
		fe.Stop(deleteCtx)
	}()
	startError := func(err error) {
		logrus.Errorf("FE error: %v", err)
		cancel()
	}
	exitOnErrCh(ctx, startError, feErrCh)

	// monitor routing sessions
	monErrCh := make(chan error, 1)
	defer close(monErrCh)
	fe.Monitor(ctx, monErrCh)
	monitorError := func(err error) {
		logrus.Errorf("FE monitor error: %v", err)
		cancel()
	}
	exitOnErrCh(ctx, monitorError, monErrCh)

	// start watching events of interest via NSP
	watchError := func(err error) {
		logrus.Errorf("FE config watcher error: %v", err)
		cancel()
	}
	go watchConfig(ctx, watchError, c, fe)

	<-ctx.Done()
	logrus.Infof("FE shutting down")
}

type onError func(err error)

func exitOnErrCh(ctx context.Context, efn onError, errCh <-chan error) {
	// If we already have an error, log it and exit
	select {
	case err, ok := <-errCh:
		if ok {
			efn(err)
		}
	default:
	}
	// Otherwise wait for an error in the background to log and cancel
	go func(ctx context.Context, errCh <-chan error) {
		select {
		case <-ctx.Done():
			logrus.Debugf("exitOnErrCh: context closed")
		case err, ok := <-errCh:
			if ok {
				efn(err)
			}
		}
	}(ctx, errCh)
}

func watchConfig(ctx context.Context, efn onError, c *feConfig.Config, fe *frontend.FrontEndService) {
	if err := fe.WaitStart(ctx); err != nil {
		efn(err)
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
		efn(fmt.Errorf("cannot watch Attractor: %v", err))
	}
}

func watchAttractor(ctx context.Context, cli nspAPI.ConfigurationManagerClient, toWatch *nspAPI.Attractor, fe *frontend.FrontEndService) error {
	watchAttractor, err := cli.WatchAttractor(ctx, toWatch)
	if err != nil {
		return err
	}
	for {
		attractorResponse, err := watchAttractor.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Infof("Attractor watcher closing down")
			return err
		}
		logrus.Infof("Attractor change event")
		if err := fe.SetNewConfig(ctx, attractorResponse.Attractors); err != nil {
			return err
		}
	}
	return nil
}
