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
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/configuration/manager"
	"github.com/nordix/meridio/pkg/configuration/monitor"
	"github.com/nordix/meridio/pkg/configuration/registry"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/nsp"
	targetRegistry "github.com/nordix/meridio/pkg/nsp/registry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

	var config Config
	err := envconfig.Process("nsp", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	healthChecker, err := health.NewChecker(8000)
	if err != nil {
		logrus.Fatalf("Unable to create Health checker: %v", err)
	}
	go func() {
		err := healthChecker.Start()
		if err != nil {
			logrus.Fatalf("Unable to start Health checker: %v", err)
		}
	}()

	// configuration
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationMonitor, err := monitor.New(config.ConfigMapName, config.Namespace, configurationRegistry)
	if err != nil {
		logrus.Fatalf("Unable to start configuration monitor: %v", err)
	}
	go configurationMonitor.Start(context.Background())
	watcherNotifier := manager.NewWatcherNotifier(configurationRegistry, configurationEventChan)
	go watcherNotifier.Start(context.Background())
	configurationManagerServer := manager.NewServer(watcherNotifier)

	// target registry
	targetRegistryEventChan := make(chan struct{}, 10)
	tr := targetRegistry.New(targetRegistryEventChan)
	watcherNotifierTargetRegistry := nsp.NewWatcherNotifier(tr, targetRegistryEventChan)
	go watcherNotifierTargetRegistry.Start(context.Background())
	targetRegistryServer := nsp.NewServer(tr, watcherNotifierTargetRegistry)

	server := grpc.NewServer()
	nspAPI.RegisterTargetRegistryServer(server, targetRegistryServer)
	nspAPI.RegisterConfigurationManagerServer(server, configurationManagerServer)

	listener, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", config.Port))
	if err != nil {
		logrus.Fatalf("NSP Service: failed to listen: %v", err)
	}

	if err := server.Serve(listener); err != nil {
		logrus.Errorf("NSP Service: failed to serve: %v", err)
	}

	<-ctx.Done()
}
