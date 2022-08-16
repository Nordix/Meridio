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
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/cmd/proxy/internal/config"
	"github.com/nordix/meridio/cmd/proxy/internal/service"
	endpointOld "github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
	"github.com/nordix/meridio/pkg/health/probe"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/nordix/meridio/pkg/retry"

	"github.com/nordix/meridio/pkg/proxy"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)

	var config config.Config
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	logrus.Infof("rootConf: %+v", config)
	if err := config.IsValid(); err != nil {
		logrus.Fatalf("%v", err)
	}
	ctx = setLogging(ctx, &config)

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.ProxyReadinessServices...); err != nil {
		logrus.Warnf("%v", err)
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
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}

	// connect IPAM the proxy relies on to assign IPs both locally and remote via nsc and nse
	logrus.Infof("Dial IPAM (%v)", config.IPAMService)
	conn, err := grpc.DialContext(signalCtx,
		config.IPAMService,
		grpc.WithTransportCredentials(
			credentials.GetClient(ctx),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Fatalf("Error dialing IPAM: %+v", err)
	}
	defer conn.Close()

	// monitor status of IPAM connection and adjust probe status accordingly
	if err := connection.Monitor(signalCtx, health.IPAMCliSvc, conn); err != nil {
		logrus.Warnf("IPAM connection state monitor err: %v", err)
	}

	ipamClient := ipamAPI.NewIpamClient(conn)
	conduit := &nspAPI.Conduit{
		Name: config.Conduit,
		Trench: &nspAPI.Trench{
			Name: config.Trench,
		},
	}
	p := proxy.NewProxy(conduit, config.Host, ipamClient, config.IPFamily, netUtils)

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)
	defer nsmAPIClient.Delete()

	// create and start NSC that connects all remote NSE belonging to the right service
	interfaceMonitorClient := interfacemonitor.NewClient(interfaceMonitor, p, netUtils)
	nsmClient := service.GetNSC(ctx, &config, nsmAPIClient, p, interfaceMonitorClient)
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
	}
	interfaceMonitorServer := interfacemonitor.NewServer(interfaceMonitor, p, netUtils)
	ep := service.StartNSE(ctx, endpointConfig, nsmAPIClient, p, interfaceMonitorServer)
	defer func() {
		deleteCtx, deleteClose := context.WithTimeout(ctx, 3*time.Second)
		defer deleteClose()
		if err := ep.Delete(deleteCtx); err != nil {
			logrus.Errorf("Err delete NSE: %v", err)
		}
	}()
	// internal probe checking health of NSE
	probe.CreateAndRunGRPCHealthProbe(signalCtx, health.NSMEndpointSvc, probe.WithAddress(ep.Server.GetUrl()), probe.WithSpiffe())

	// connect NSP and start watching config events of interest
	configurationContext, configurationCancel := context.WithCancel(signalCtx)
	defer configurationCancel()
	logrus.Infof("Dial NSP (%v)", nsp.GetService(config.NSPServiceName, config.Trench, config.Namespace, config.NSPServicePort))
	nspConn, err := grpc.DialContext(signalCtx,
		nsp.GetService(config.NSPServiceName, config.Trench, config.Namespace, config.NSPServicePort),
		grpc.WithTransportCredentials(
			credentials.GetClient(configurationContext),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Fatalf("Error dialing NSP: %v", err)
	}
	defer nspConn.Close()

	// monitor status of NSP connection and adjust probe status accordingly
	if err := connection.Monitor(signalCtx, health.NSPCliSvc, nspConn); err != nil {
		logrus.Warnf("NSP connection state monitor err: %v", err)
	}

	configurationManagerClient := nspAPI.NewConfigurationManagerClient(nspConn)
	if err != nil {
		logrus.Fatalf("WatchVip err: %v", err)
	}

	logrus.Debugf("Watch configuration")
	err = retry.Do(func() error {
		vipWatcher, err := configurationManagerClient.WatchVip(configurationContext, &nspAPI.Vip{
			Trench: &nspAPI.Trench{
				Name: config.Trench,
			},
		})
		if err != nil {
			return err
		}
		for {
			vipResponse, err := vipWatcher.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			p.SetVIPs(vipResponse.ToSlice())
		}
		return nil
	}, retry.WithContext(signalCtx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		logrus.Errorf("WatchVip err: %v", err)
	}
	logrus.Infof("Shutting done...")
}

func setLogging(ctx context.Context,
	config *config.Config) context.Context {
	logrus.SetLevel(func() logrus.Level {
		l, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			logrus.Fatalf("invalid log level %s", config.LogLevel)
		}
		if l == logrus.TraceLevel {
			log.EnableTracing(true) // enable tracing in NSM
		}
		return l
	}())

	return log.WithLog(ctx, logruslogger.New(ctx)) // allow NSM logs
}
