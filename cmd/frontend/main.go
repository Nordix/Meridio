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

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/cmd/frontend/internal/env"
	"github.com/nordix/meridio/cmd/frontend/internal/frontend"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/security/credentials"
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
	logrus.SetFormatter(&nested.Formatter{})

	config := &env.Config{}
	if err := envconfig.Usage("nfe", config); err != nil {
		logrus.Fatal(err)
	}
	if err := envconfig.Process("nfe", config); err != nil {
		logrus.Fatalf("%v", err)
	}
	logrus.Infof("rootConf: %+v", config)

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.FeReadinessServices...); err != nil {
		logrus.Warnf("%v", err)
	}

	fe := frontend.NewFrontEndService(config)
	defer fe.CleanUp()
	health.SetServingStatus(ctx, health.TargetRegistryCliSvc, true) // NewFrontEndService() creates Target Registry Client

	/* if err := fe.AddVIPRules(); err != nil {
		cancel()
		logrus.Fatalf("Failed to setup src routes for VIPs: %v", err)
	} */

	if err := fe.Init(); err != nil {
		cancel()
		logrus.Fatalf("Init failed: %v", err)
	}

	feErrCh := fe.Start(ctx)
	exitOnErrCh(ctx, cancel, feErrCh)

	/* time.Sleep(1 * time.Second)
	if err := fe.VerifyConfig(ctx); err != nil {
		logrus.Errorf("Failed to verify config")
	} */

	// monitor BIRD routing sessions
	if err := fe.Monitor(ctx); err != nil {
		cancel()
		logrus.Fatalf("Failed to start monitor: %v", err)
	}

	go watchConfig(ctx, cancel, config, fe)

	<-ctx.Done()
	logrus.Warnf("FE shutting down")
}

func exitOnErrCh(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) {
	// If we already have an error, log it and exit
	select {
	case err, ok := <-errCh:
		if ok {
			logrus.Errorf("exitOnErrCh(0): %v", err)
		}
	default:
	}
	// Otherwise wait for an error in the background to log and cancel
	go func(ctx context.Context, errCh <-chan error) {
		if err, ok := <-errCh; ok {
			logrus.Errorf("exitOnErrCh(1): %v", err)
		}
		cancel()
	}(ctx, errCh)
}

func watchConfig(ctx context.Context, cancel context.CancelFunc, c *env.Config, fe *frontend.FrontEndService) {
	if err := fe.WaitStart(ctx); err != nil {
		logrus.Errorf("Wait start: %v", err)
		cancel()
	}
	conn, err := grpc.Dial(c.NSPService,
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Errorf("grpc.Dial err: %v", err)
		cancel()
	}
	health.SetServingStatus(ctx, health.NSPCliSvc, true)
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(conn)
	attractorToWatch := &nspAPI.Attractor{
		Name: c.AttractorName,
		Trench: &nspAPI.Trench{
			Name: c.TrenchName,
		},
	}
	if err := watchAttractor(ctx, configurationManagerClient, attractorToWatch, fe); err != nil {
		logrus.Errorf("Attractor watcher: %v", err)
		cancel()
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
		logrus.Infof("Attractor config change event")
		if err := fe.SetNewConfig(attractorResponse.Attractors); err != nil {
			return err
		}
	}
	return nil
}
