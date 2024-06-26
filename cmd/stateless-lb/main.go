/*
Copyright (c) 2021-2022 Nordix Foundation
Copyright (c) 2024 OpenInfra Foundation Europe

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
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	nsmlog "github.com/networkservicemesh/sdk/pkg/tools/log"
	lbAPI "github.com/nordix/meridio/api/loadbalancer/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/debug"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/connection"
	"github.com/nordix/meridio/pkg/health/probe"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/kernel/neighbor"
	"github.com/nordix/meridio/pkg/loadbalancer/flow"
	"github.com/nordix/meridio/pkg/loadbalancer/nfqlb"
	"github.com/nordix/meridio/pkg/loadbalancer/stream"
	"github.com/nordix/meridio/pkg/loadbalancer/target"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/metrics"
	"github.com/nordix/meridio/pkg/nat"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	nsmmetrics "github.com/nordix/meridio/pkg/nsm/metrics"
	nsmmonitor "github.com/nordix/meridio/pkg/nsm/monitor"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/nordix/meridio/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

func printHelp() {
	fmt.Println(`
stateless-lb --
  The stateless-lb process in https://github.com/Nordix/Meridio
  sets up load-balancing using https://github.com/Nordix/nfqueue-loadbalancer.
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
	err := envconfig.Process("nsm", &config)
	if err != nil {
		panic(err)
	}
	logger := log.New("Meridio-LB", config.LogLevel)
	logger.Info("Configuration read", "config", config)

	ctx, cancel := signal.NotifyContext(
		logr.NewContext(context.Background(), logger),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()
	if err := config.IsValid(); err != nil {
		log.Fatal(logger, "invalid config", "error", err)
	}

	if config.LogLevel == "TRACE" {
		nsmlog.EnableTracing(true)
		// Work-around for hard-coded logrus dependency in NSM
		logrus.SetLevel(logrus.TraceLevel)
	}
	logger.Info("NSM trace", "enabled", nsmlog.IsTracingEnabled())
	// See https://github.com/networkservicemesh/sdk/issues/1272
	nsmlogger := log.NSMLogger(logger)
	nsmlog.SetGlobalLogger(nsmlogger)
	ctx = nsmlog.WithLog(ctx, nsmlogger)

	netUtils := &linuxKernel.KernelUtils{}

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinessSubservices(ctx, health.LBReadinessServices...); err != nil {
		logger.Error(err, "RegisterReadinessSubservices")
	}
	// note: NSM endpoint service is hosted from early on by its server, thus it can be probed
	// irrespective of its registration status at NSM
	if err := health.RegisterLivenessSubservices(ctx, health.LBLivenessServices...); err != nil {
		logger.Error(err, "RegisterLivenessSubservices")
	}

	logger.Info("Dial NSP", "NSPService", config.NSPService)
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
		log.Fatal(logger, "Dial NSP", "error", err)
	}
	defer conn.Close()

	// monitor status of NSP connection and adjust probe status accordingly
	if err := connection.Monitor(ctx, health.NSPCliSvc, conn); err != nil {
		logger.Error(err, "NSP connection state monitor")
	}

	stream.SetInterfaceNamePrefix(config.ServiceName) // deduce the NSM interfacename prefix for the netfilter defrag rules
	targetRegistryClient := nspAPI.NewTargetRegistryClient(conn)
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(conn)
	conduit := &nspAPI.Conduit{
		Name: config.ConduitName,
		Trench: &nspAPI.Trench{
			Name: config.TrenchName,
		},
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(logger, "Unable to get hostname", "error", err)
	}

	targetHitsMetrics, err := target.NewTargetHitsMetrics()
	if err != nil {
		log.Fatal(logger, "Unable to init lb target metrics", "error", err)
	}

	interfaceMetrics := linuxKernel.NewInterfaceMetrics([]metric.ObserveOption{
		metric.WithAttributes(attribute.String("trench", config.TrenchName)),
		metric.WithAttributes(attribute.String("conduit", config.ConduitName)),
	})

	lbFactory := nfqlb.NewLbFactory(nfqlb.WithNFQueue(config.Nfqueue))
	nfa, err := nfqlb.NewNetfilterAdaptor(nfqlb.WithNFQueue(config.Nfqueue), nfqlb.WithNFQueueFanout(config.NfqueueFanout))
	if err != nil {
		logger.Error(err, "netfilter adaptor create")
		cancel()
		return
	}
	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logger.Error(err, "interface monitor create")
		cancel()
		return
	}
	neighborMonitor, err := neighbor.NewNeighborMonitor(ctx, neighbor.WithStateMask(netlink.NUD_FAILED|netlink.NUD_REACHABLE))
	if err != nil {
		logger.Error(err, "neighbor monitor create")
		cancel()
		return
	}

	// start server to host Stream Forwarding Availability service
	lis, err := createStreamAvailabilityListener(config)
	if err != nil {
		logger.Error(err, "createStreamAvailabilityListener")
		cancel()
		return
	}
	s := grpc.NewServer(
		grpc.Creds(credentials.GetServer(context.Background())),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime: config.GRPCKeepaliveTime,
		}),
	)
	defer func() {
		// attempt graceful shutdown to allow sending out pending msgs
		stopped := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(stopped)
		}()
		waitTimer := time.NewTimer(time.Second)
		select {
		case <-waitTimer.C:
			s.Stop() // graceful shutdown not finished in time, force stop immediately
		case <-stopped:
			waitTimer.Stop()
			select {
			case <-waitTimer.C:
			default:
			}
		}
	}()
	// announces forwarding availability of streams (i.e. if the LB can forward traffic towards application targets)
	streamFwdAvailabilityService := stream.NewForwardingAvailabilityService(
		context.Background(),
		&lbAPI.Target{
			Context: map[string]string{
				types.IdentifierKey: hostname,
			},
		},
	)
	defer streamFwdAvailabilityService.Stop()
	lbAPI.RegisterStreamAvailabilityServiceServer(s, streamFwdAvailabilityService)
	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Error(err, "Failed to serve on lb socket")
		}
	}()

	sns := newSimpleNetworkService(
		netUtils.WithInterfaceMonitor(ctx, interfaceMonitor),
		targetRegistryClient,
		configurationManagerClient,
		conduit,
		netUtils,
		lbFactory, // to spawn nfqlb instance for each Stream created
		nfa,       // netfilter kernel configuration to steer VIP traffic to nfqlb process
		config.IdentifierOffsetStart,
		targetHitsMetrics,
		neighborMonitor,
		streamFwdAvailabilityService,
	)

	interfaceMonitorEndpoint := interfacemonitor.NewServer(interfaceMonitor, sns, netUtils)

	// Note: naming the interface is left to NSM (refer to getNameFromConnection())
	// However NSM does not seem to ensure uniqueness either. Might need to revisit...
	responderEndpoint := []networkservice.NetworkServiceServer{
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfaceMonitorEndpoint,
		nsmmetrics.NewServer(interfaceMetrics),
		sendfd.NewServer(),
	}

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
		GRPCMaxBackoff:   config.GRPCMaxBackoff,
	}
	nsmAPIClient := nsm.NewAPIClient(context.Background(), apiClientConfig) // background context to allow endpoint unregistration on tear down
	defer nsmAPIClient.Delete()

	// connect NSMgr and start NSM connection monitoring (to log events of interest)
	cc, err := grpc.DialContext(ctx,
		grpcutils.URLToTarget(&nsmAPIClient.Config.ConnectTo),
		nsmAPIClient.GRPCDialOption...,
	)
	if err != nil {
		logger.Error(err, "Dialing NSMgr")
		cancel()
		return
	}
	defer cc.Close()
	monitorClient := networkservice.NewMonitorConnectionClient(cc)
	go nsmmonitor.ConnectionMonitor(ctx, config.Name, monitorClient)

	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		Labels:           make(map[string]string),
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	ep, err := endpoint.NewEndpoint(
		context.Background(), // use background context to allow endpoint unregistration on tear down
		endpointConfig,
		nsmAPIClient.NetworkServiceRegistryClient,
		nsmAPIClient.NetworkServiceEndpointRegistryClient,
		responderEndpoint...,
	)
	if err != nil {
		log.Fatal(logger, "Unable to create a new NSE", "error", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*3))
		defer cancel()
		ep.Delete(ctx) // let endpoint unregister with NSM to inform proxies in time
		logger.Info("LB endpoint deleted")
	}()

	probe.CreateAndRunGRPCHealthProbe(
		ctx,
		health.NSMEndpointSvc,
		probe.WithAddress(ep.GetUrl()),
		probe.WithSpiffe(),
		probe.WithRPCTimeout(config.GRPCProbeRPCTimeout),
	)

	go func() {
		logger.Info("Start nfqlb process")
		if err := lbFactory.Start(ctx); err != nil && ctx.Err() != context.Canceled {
			logger.Error(err, "Failure running nfqlb process")
		}
		logger.Info("Process nfqlb terminated")
		cancel()
	}()
	sns.Start()
	// monitor availibilty of frontends (advertise NSE to proxies only if there's feasible FE)
	fns := NewFrontendNetworkService(ctx, targetRegistryClient, ep, NewServiceControlDispatcher(sns))
	go func() {
		logger.Info("Start frontend monitoring service")
		if err := fns.Start(); err != nil && status.Code(err) != codes.Canceled {
			logger.Error(err, "Frontend monitoring")
		}
		logger.Info("Frontend monitoring service terminated")
		cancel()
	}()

	if config.MetricsEnabled {
		func() {
			_, err = metrics.Init(ctx)
			if err != nil {
				logger.Error(err, "Unable to init metrics collector")
				cancel()
				return
			}

			err = flow.CollectMetrics(
				flow.WithTrenchName(config.TrenchName),
				flow.WithConduitName(config.ConduitName),
			)
			if err != nil {
				logger.Error(err, "Unable to start flow metrics collector")
				cancel()
				return
			}

			err = targetHitsMetrics.Collect()
			if err != nil {
				logger.Error(err, "Unable to start target hits metrics collector")
				cancel()
				return
			}

			err = interfaceMetrics.Collect()
			if err != nil {
				logger.Error(err, "Unable to start interface metrics collector")
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
}

func createStreamAvailabilityListener(config Config) (net.Listener, error) {
	address := ""
	switch config.Socket.Scheme {
	case "unix":
		address = config.Socket.Path
		if address == "" {
			address = config.Socket.Opaque
		}
		if err := os.RemoveAll(address); err != nil {
			return nil, fmt.Errorf("failed removing unix socket: %w", err)
		}
	case "tcp":
		address = config.Socket.Host
	}
	lis, err := net.Listen(config.Socket.Scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed listen on socket: %w", err)
	}
	if config.Socket.Scheme == "unix" {
		if err := os.Chmod(address, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed changing unix socket permission to %v: %w", os.ModePerm, err)
		}
	}
	return lis, nil
}

// SimpleNetworkService -
type SimpleNetworkService struct {
	*nspAPI.Conduit
	targetRegistryClient         nspAPI.TargetRegistryClient
	ConfigurationManagerClient   nspAPI.ConfigurationManagerClient
	IdentifierOffsetGenerator    *IdentifierOffsetGenerator
	interfaces                   sync.Map
	ctx                          context.Context
	logger                       logr.Logger
	streams                      map[string]types.Stream
	netUtils                     networking.Utils
	nfqueueIndex                 int
	serviceCtrCh                 chan bool
	simpleNetworkServiceBlocked  bool
	mu                           sync.Mutex
	cancelStreamWatcher          context.CancelFunc
	streamWatcherRunning         bool
	lbFactory                    types.NFQueueLoadBalancerFactory
	nfa                          types.NFAdaptor
	natHandler                   *nat.NatHandler
	targetHitsMetrics            *target.HitsMetrics
	neighborMonitor              *neighbor.NeighborMonitor
	streamFwdAvailabilityService *stream.ForwardingAvailabilityService
}

/* // Request checks if allowed to serve the request
// A non-nil error is returned if serving the request was rejected, or if a next element in the chain returns a non-nil error
// It implements NetworkServiceServer for SimpleNetworkService
//
// TODO: Is this feature even needed? Currently, SimpleNetworkServiceClient will keep trying to establish an NSM connection
// forever, during which it also blocks NSE event processing. So it won't notice if the NSE has disappeared in the meantime.
// Although this is a valid problem, irrespective of the fact whether SimpleNetworkService blocks Requests or not...
// Moreover generally NSM is really pushing to establish a connection on Requests, thus letting the Request through, could lead
// to a better outcome...
func (sns *SimpleNetworkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if sns.serviceBlocked() {
		return nil, errors.New("SimpleNetworkService blocked")
	}
	logrus.Infof("SimpleNetworkService: Request")
	return next.Server(ctx).Request(ctx, request)
}
// Close it does nothing except calling the next Close in the chain
// A non-nil error if a next element in the chain returns a non-nil error
// It implements NetworkServiceServer for SimpleNetworkService
func (sns *SimpleNetworkService) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logrus.Infof("SimpleNetworkService: Close")
	return next.Server(ctx).Close(ctx, conn)
} */

// NewSimpleNetworkService -
func newSimpleNetworkService(
	ctx context.Context,
	targetRegistryClient nspAPI.TargetRegistryClient,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	conduit *nspAPI.Conduit,
	netUtils networking.Utils,
	lbFactory types.NFQueueLoadBalancerFactory,
	nfa types.NFAdaptor,
	identifierOffsetStart int,
	targetHitsMetrics *target.HitsMetrics,
	neighborMonitor *neighbor.NeighborMonitor,
	streamFwdAvailabilityService *stream.ForwardingAvailabilityService,
) *SimpleNetworkService {
	identifierOffsetGenerator := NewIdentifierOffsetGenerator(identifierOffsetStart)
	logger := log.FromContextOrGlobal(ctx).WithValues("class", "SimpleNetworkService",
		"conduit", conduit,
	)
	nh, err := nat.NewNatHandler()
	if err != nil {
		log.Fatal(logger, "Unable to init NAT", "error", err)
	}
	simpleNetworkService := &SimpleNetworkService{
		Conduit:                      conduit,
		targetRegistryClient:         targetRegistryClient,
		ConfigurationManagerClient:   configurationManagerClient,
		IdentifierOffsetGenerator:    identifierOffsetGenerator,
		ctx:                          ctx,
		logger:                       logger,
		netUtils:                     netUtils,
		nfqueueIndex:                 1,
		streams:                      make(map[string]types.Stream),
		serviceCtrCh:                 make(chan bool),
		simpleNetworkServiceBlocked:  true,
		streamWatcherRunning:         false,
		lbFactory:                    lbFactory,
		nfa:                          nfa,
		natHandler:                   nh,
		targetHitsMetrics:            targetHitsMetrics,
		neighborMonitor:              neighborMonitor,
		streamFwdAvailabilityService: streamFwdAvailabilityService,
	}
	logger.Info("Created LB service", "conduit", conduit)
	return simpleNetworkService
}

// Start -
func (sns *SimpleNetworkService) Start() {
	go sns.watchVips(sns.ctx)
	go sns.watchConduit(sns.ctx)
	go func() {
		sns.logger.Info("Watch LB service control channel")
		defer sns.logger.Info("Stopped watching LB service control channel")
		for {
			select {
			case allowService, ok := <-sns.serviceCtrCh:
				if ok {
					sns.logger.Info("LB service control event", "allowService", allowService)
					sns.mu.Lock()

					sns.simpleNetworkServiceBlocked = !allowService
					// When service is blocked it implies that NSE facing the proxies gets also removed.
					// Removal of the NSE from registry prompts the NSC side to close the related NSM
					// connections making the associated interfaces unusable. However unfortunately
					// NSM is not able to properly close a connection associated with a "disappeared" NSE
					// (so NSM interfaces remain as well).
					//
					// Thus in SimpleNetworkService we must prohibit processing of new Targets and
					// creation of new NSE interfaces connecting proxies while NSE removal takes effect
					// on NSC side.
					// Moreover the known Targets and thus the associated routing must be force removed.
					// That's because once the "block" is lifted, the NSE serving proxies should be advertised
					// again, resulting in new NSM Service Requests and thus interfaces for which the Target
					// routes must be readjusted.
					// Interference of old NSM interfaces must be avoided, thus their link state is changed
					// to down. (Hopefully once NSM finally decides to remove an old interface (e.g. due
					// to some timeout or whatever) this state change won't screw up things...)
					if sns.simpleNetworkServiceBlocked {
						sns.evictStreams()
						sns.disableInterfaces()
						if sns.cancelStreamWatcher != nil {
							sns.cancelStreamWatcher()
							sns.streamWatcherRunning = false
						}
					} else {
						// restart watching the streams
						sns.startStreamWatcher()
					}
					sns.mu.Unlock()
				}
			case <-sns.ctx.Done():
				return
			}
		}
	}()
}

func (sns *SimpleNetworkService) startStreamWatcher() {
	if sns.streamWatcherRunning {
		return
	}
	sns.streamWatcherRunning = true
	var ctx context.Context
	ctx, sns.cancelStreamWatcher = context.WithCancel(sns.ctx)
	go func() {
		err := retry.Do(func() error {
			return sns.watchStreams(ctx)
		}, retry.WithContext(ctx),
			retry.WithDelay(500*time.Millisecond),
			retry.WithErrorIngnored())
		if err != nil && status.Code(err) != codes.Canceled {
			sns.logger.Error(err, "watchStreams")
		}
	}()
}

// InterfaceCreated -
func (sns *SimpleNetworkService) InterfaceCreated(intf networking.Iface) {
	if len(intf.GetLocalPrefixes()) < 1 {
		// ignore pure netlink based monitor notification
		return
	}
	_ = intf.GetName() // fills the Name field of the interface if necessary
	if _, ok := sns.interfaces.Load(intf.GetIndex()); !ok {
		sns.logger.Info("InterfaceCreated", "interface", intf)
	}
	if sns.serviceBlocked() {
		// if service blocked, do not process new interface events (which
		// might appear until the block takes effect on NSC side)
		// instead disable them not to interfere after the block is lifted
		sns.disableInterface(intf)
		return
	}
	// https://github.com/Nordix/Meridio/issues/392
	// The LB may get double interfaces to the same proxy. The new one works,
	// the older must be disabled
	sns.interfaces.Range(func(key, value any) bool {
		if key == intf.GetIndex() {
			return true // Possibe? (better safe than sorry)
		}
		oldif := value.(networking.Iface)
		if sameSubnet(intf, oldif) {
			sns.logger.Info("Interface replaced during interface create event", "old", oldif, "new", intf)
			sns.disableInterface(oldif)
			// remove replaced interface from the list in order to avoid further
			// unnecessary and confusing replace printouts and disable attempts
			// in case the old interface lingers on
			sns.interfaces.Delete(oldif.GetIndex())
		}
		return true
	})
	sns.interfaces.Store(intf.GetIndex(), intf)
}

// sameSubnet Returns true if interfaces uses the same subnet.
func sameSubnet(if1, if2 networking.Iface) bool {
	cidrs1 := if1.GetLocalPrefixes()
	if len(cidrs1) < 1 {
		return false // Possibe? (better safe than sorry)
	}
	// It is enough to check either ipv4 or ipv6.
	_, net1, err := net.ParseCIDR(cidrs1[0])
	if err != nil {
		return false // Shouldn't happen
	}
	for _, cidr2 := range if2.GetLocalPrefixes() {
		_, net2, err := net.ParseCIDR(cidr2)
		if err != nil {
			return false // Shouldn't happen
		}
		if net1.IP.Equal(net2.IP) {
			return true
		}
	}
	return false
}

// InterfaceDeleted -
func (sns *SimpleNetworkService) InterfaceDeleted(intf networking.Iface) {
	sns.logger.Info("InterfaceDeleted", "interface", intf)
	sns.interfaces.Delete(intf.GetIndex())
}

func (sns *SimpleNetworkService) watchStreams(ctx context.Context) error {
	streamsToWatch := &nspAPI.Stream{
		Conduit: sns.Conduit,
	}
	watchStream, err := sns.ConfigurationManagerClient.WatchStream(ctx, streamsToWatch)
	if err != nil {
		return fmt.Errorf("failed to create stream watcher (%s): %w", streamsToWatch.String(), err)
	}
	for {
		streamResponse, err := watchStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream watcher receive error (%s): %w", streamsToWatch.String(), err)
		}
		err = sns.updateStreams(streamResponse.Streams)
		if err != nil {
			sns.logger.Error(err, "updateStreams")
		}
	}
	return nil
}

func (sns *SimpleNetworkService) updateStreams(streams []*nspAPI.Stream) error {
	remainingStreams := make(map[string]struct{})
	for streamName := range sns.streams {
		remainingStreams[streamName] = struct{}{}
	}
	var errFinal error
	for _, s := range streams {
		// check if stream belongs to this conduit and trench
		if !sns.Conduit.Equals(s.GetConduit()) {
			continue
		}
		_, exists := sns.streams[s.GetName()]
		if !exists { // todo: create a stream
			err := sns.addStream(s)
			if err != nil {
				errFinal = utils.AppendErr(errFinal, fmt.Errorf("addStream failed during update (%s): %w",
					s.GetName(), err)) // todo
			}
		} else { // todo: check if need an update
			delete(remainingStreams, s.GetName())
		}
	}
	// remove remaining ones
	for streamName := range remainingStreams {
		err := sns.deleteStream(streamName)
		if err != nil {
			errFinal = utils.AppendErr(errFinal, fmt.Errorf("deleteStream failed during update (%s): %w",
				streamName, err)) // todo

		}
	}
	// adjust stream service serving status (needs at least 1 stream)
	health.SetServingStatus(sns.ctx, health.StreamSvc, len(sns.streams) > 0)
	return errFinal
}

func (sns *SimpleNetworkService) addStream(strm *nspAPI.Stream) error {
	// verify if stream belongs to this conduit and trench
	logger := sns.logger.WithValues("func", "addStream", "stream", strm.String())
	_, exists := sns.streams[strm.GetName()]
	if exists {
		return errors.New("this stream already exists")
	}
	identifierOffset, err := sns.IdentifierOffsetGenerator.Generate(strm)
	if exists {
		return fmt.Errorf("failed to generate identifier offset when adding stream (%s): %w", strm.String(), err)
	}
	neighborReachDetector, err := neighbor.NewNeighborReachabilityDetector(sns.ctx, strm.GetName(), sns.neighborMonitor)
	if err != nil {
		return fmt.Errorf("failed to create neighbor cache for stream (%s): %w", strm.String(), err)
	}
	logger.Info("Create LB stream")
	s, err := stream.New(
		strm,
		sns.targetRegistryClient,
		sns.ConfigurationManagerClient,
		sns.nfqueueIndex,
		sns.netUtils,
		sns.lbFactory,
		identifierOffset,
		sns.targetHitsMetrics,
		neighborReachDetector,
		sns.streamFwdAvailabilityService,
	)
	if err != nil {
		return fmt.Errorf("failed to create stream (%s): %w", strm.String(), err)
	}
	go func() {
		logger.Info("Start LB stream")
		defer logger.Info("LB stream terminated")
		err := s.Start(sns.ctx)
		if err != nil && status.Code(err) != codes.Canceled {
			logger.Error(err, "running LB stream")
		}
		neighborReachDetector.Close()
	}()
	sns.nfqueueIndex = sns.nfqueueIndex + 1
	sns.streams[strm.GetName()] = s
	return nil
}

func (sns *SimpleNetworkService) deleteStream(streamName string) error {
	// verify if stream belongs to this conduit and trench
	stream, exists := sns.streams[streamName]
	if !exists {
		return nil
	}
	sns.IdentifierOffsetGenerator.Release(streamName)
	err := stream.Delete()
	delete(sns.streams, streamName)
	if err != nil {
		return fmt.Errorf("failed to delete stream (%s): %w", streamName, err)
	}
	return nil
}

func (sns *SimpleNetworkService) serviceBlocked() bool {
	sns.mu.Lock()
	defer sns.mu.Unlock()
	return sns.simpleNetworkServiceBlocked
}

func (sns *SimpleNetworkService) GetServiceControlChannel() interface{} {
	return (chan<- bool)(sns.serviceCtrCh)
}

func (sns *SimpleNetworkService) evictStreams() {
	sns.logger.Info("Evict streams")
	for name, stream := range sns.streams {
		err := stream.Delete()
		if err != nil {
			sns.logger.Error(err, "stream delete", "stream", name)
		}
	}
	sns.streams = make(map[string]types.Stream)
}

// disableInterfaces -
// Set interfaces down, so that they won't interfere with future "Add Target"
// operations. Meaning old interfaces not yet removed by NSM must not get
// associated with routes inserted for Targets after the block is lifted.
func (sns *SimpleNetworkService) disableInterfaces() {
	sns.logger.Info("Disable interfaces", "func", "disableInterfaces")
	sns.interfaces.Range(func(key interface{}, value interface{}) bool {
		sns.disableInterface(value.(networking.Iface))
		sns.interfaces.Delete(key)
		return true
	})
}

// disableInterface -
// Set interface state down
func (sns *SimpleNetworkService) disableInterface(intf networking.Iface) {
	sns.logger.V(1).Info("Disable interface", "func", "disableInterface", "interface", intf)
	la := netlink.NewLinkAttrs()
	la.Index = intf.GetIndex()
	err := netlink.LinkSetDown(&netlink.Dummy{LinkAttrs: la})
	if err != nil {
		sns.logger.Error(err, "LinkSetDown", "interface", intf)
	}
}

// watchVips -
// Monitors VIP changes in Trench via NSP
func (sns *SimpleNetworkService) watchVips(ctx context.Context) {
	logger := sns.logger.WithValues("func", "watchVips")
	logger.Info("Watch VIPs")
	defer logger.Info("Stopped watching VIPs")

	err := retry.Do(func() error {
		vipsToWatch := &nspAPI.Vip{
			Trench: sns.Conduit.GetTrench(),
		}
		watchVip, err := sns.ConfigurationManagerClient.WatchVip(ctx, vipsToWatch)
		if err != nil {
			return fmt.Errorf("failed to create vip watcher (%s): %w", vipsToWatch.String(), err)
		}
		for {
			vipResponse, err := watchVip.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("vip watcher receive error (%s): %w", vipsToWatch.String(), err)
			}
			err = sns.updateVips(vipResponse.GetVips())
			if err != nil {
				logger.Error(err, "updateVips")
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())

	if err != nil && status.Code(err) != codes.Canceled {
		logger.Error(err, "watchVIPs") // todo
	}
}

// watchConduit -
func (sns *SimpleNetworkService) watchConduit(ctx context.Context) {
	logger := sns.logger.WithValues("func", "watchConduit", "conduit", sns.Conduit)
	logger.Info("Start watching conduit")
	defer logger.Info("Stopped watching conduit")

	err := retry.Do(func() error {
		conduitToWatch := sns.Conduit
		watchConduit, err := sns.ConfigurationManagerClient.WatchConduit(ctx, conduitToWatch)
		if err != nil {
			return fmt.Errorf("failed to create conduit watcher (%s): %w", conduitToWatch.String(), err)
		}
		for {
			conduitResponse, err := watchConduit.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("conduit watcher receive error (%s): %w", conduitToWatch.String(), err)
			}
			if len(conduitResponse.GetConduits()) != 1 {
				continue
			}
			conduit := conduitResponse.GetConduits()[0]
			err = sns.natHandler.SetNats(conduit.GetDestinationPortNats())
			if err != nil {
				logger.Error(err, "SetNats")
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())

	if err != nil && status.Code(err) != codes.Canceled {
		sns.logger.Error(err, "retry")
	}
}

// updateVips -
// Sends list of VIPs to Netfilter Adaptor to adjust kerner based rules
func (sns *SimpleNetworkService) updateVips(vips []*nspAPI.Vip) error {
	sns.logger.V(1).Info("Updating VIPs", "func", "updateVips", "vips", vips)
	if err := sns.nfa.SetDestinationIPs(vips); err != nil {
		return fmt.Errorf("failed to set destination IPs during update VIPs (%v): %w", vips, err)
	}
	return nil
}
