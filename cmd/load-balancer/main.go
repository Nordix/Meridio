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
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/probe"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/loadbalancer/nfqlb"
	"github.com/nordix/meridio/pkg/loadbalancer/stream"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"
)

const (
	M = 9973
	N = 100
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
	err := envconfig.Process("nsm", &config)
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	logrus.Infof("rootConf: %+v", config)
	if err := config.IsValid(); err != nil {
		logrus.Fatalf("invalid config - %v", err)
	}

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
	ctx = log.WithLog(ctx, logruslogger.New(ctx)) // allow NSM logs

	netUtils := &linuxKernel.KernelUtils{}

	// create and start health server
	ctx = health.CreateChecker(ctx)
	if err := health.RegisterReadinesSubservices(ctx, health.LbReadinessServices...); err != nil {
		logrus.Warnf("%v", err)
	}

	conn, err := grpc.Dial(config.NSPService,
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Errorf("grpc.Dial err: %v", err)
	}
	health.SetServingStatus(ctx, health.NSPCliSvc, true)
	stream.SetInterfaceNamePrefix(config.ServiceName) // deduce the NSM interfacename prefix for the netfilter defrag rules
	targetRegistryClient := nspAPI.NewTargetRegistryClient(conn)
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(conn)
	conduit := &nspAPI.Conduit{
		Name: config.ConduitName,
		Trench: &nspAPI.Trench{
			Name: config.TrenchName,
		},
	}

	lbFactory := nfqlb.NewLbFactory(nfqlb.WithNFQueue(config.Nfqueue))
	nfa, err := nfqlb.NewNetfilterAdaptor(nfqlb.WithNFQueue(config.Nfqueue), nfqlb.WithNFQueueFanout(config.NfqueueFanout))
	if err != nil {
		logrus.Fatalf("Netfilter adaptor err: %v", err)
	}
	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	sns := NewSimpleNetworkService(
		netUtils.WithInterfaceMonitor(ctx, interfaceMonitor),
		targetRegistryClient,
		configurationManagerClient,
		conduit,
		netUtils,
		lbFactory, // to spawn nfqlb instance for each Stream created
		nfa,       // netfilter kernel configuration to steer VIP traffic to nfqlb process
	)

	interfaceMonitorEndpoint := interfacemonitor.NewServer(interfaceMonitor, sns, netUtils)

	// Note: naming the interface is left to NSM (refer to getNameFromConnection())
	// However NSM does not seem to ensure uniqueness either. Might need to revisit...
	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfaceMonitorEndpoint,
		sendfd.NewServer(),
	}

	apiClientConfig := &nsm.Config{
		Name:             config.Name,
		ConnectTo:        config.ConnectTo,
		DialTimeout:      config.DialTimeout,
		RequestTimeout:   config.RequestTimeout,
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	nsmAPIClient := nsm.NewAPIClient(context.Background(), apiClientConfig) // background context to allow endpoint unregistration on tear down
	defer nsmAPIClient.Delete()

	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		Labels:           make(map[string]string),
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	ep, err := endpoint.NewEndpoint(
		context.Background(), // background context to allow endpoint unregistration on tear down
		endpointConfig,
		nsmAPIClient.NetworkServiceRegistryClient,
		nsmAPIClient.NetworkServiceEndpointRegistryClient,
	)
	if err != nil {
		logrus.Fatalf("unable to create a new nse %+v", err)
	}

	defer ep.Delete() // let endpoint unregister with NSM to inform proxies in time
	err = ep.StartWithoutRegister(responderEndpoint...)
	if err != nil {
		logrus.Fatalf("unable to start nse %+v", err)
	}

	probe.CreateAndRunGRPCHealthProbe(ctx, health.NSMEndpointSvc, probe.WithAddress(ep.GetUrl()), probe.WithSpiffe())

	ctx = lbFactory.Start(ctx) // start nfqlb process in background
	sns.Start()
	// monitor availibilty of frontends; if no feasible FE don't advertise NSE to proxies
	fns := NewFrontendNetworkService(ctx, targetRegistryClient, ep, NewServiceControlDispatcher(sns))
	go fns.Start()

	<-ctx.Done()
}

// SimpleNetworkService -
type SimpleNetworkService struct {
	*nspAPI.Conduit
	targetRegistryClient        nspAPI.TargetRegistryClient
	ConfigurationManagerClient  nspAPI.ConfigurationManagerClient
	interfaces                  sync.Map
	ctx                         context.Context
	streams                     map[string]types.Stream
	netUtils                    networking.Utils
	nfqueueIndex                int
	serviceCtrCh                chan bool
	simpleNetworkServiceBlocked bool
	mu                          sync.Mutex
	cancelStreamWatcher         context.CancelFunc
	streamWatcherRunning        bool
	lbFactory                   types.NFQueueLoadBalancerFactory
	nfa                         types.NFAdaptor
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
func NewSimpleNetworkService(
	ctx context.Context,
	targetRegistryClient nspAPI.TargetRegistryClient,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	conduit *nspAPI.Conduit,
	netUtils networking.Utils,
	lbFactory types.NFQueueLoadBalancerFactory,
	nfa types.NFAdaptor,
) *SimpleNetworkService {
	simpleNetworkService := &SimpleNetworkService{
		Conduit:                     conduit,
		targetRegistryClient:        targetRegistryClient,
		ConfigurationManagerClient:  configurationManagerClient,
		ctx:                         ctx,
		netUtils:                    netUtils,
		nfqueueIndex:                1,
		streams:                     make(map[string]types.Stream),
		serviceCtrCh:                make(chan bool),
		simpleNetworkServiceBlocked: true,
		streamWatcherRunning:        false,
		lbFactory:                   lbFactory,
		nfa:                         nfa,
	}
	return simpleNetworkService
}

// Start -
func (sns *SimpleNetworkService) Start() {
	go sns.watchVips(sns.ctx)
	go func() {
		for {
			select {
			case allowService, ok := <-sns.serviceCtrCh:
				if ok {
					sns.mu.Lock()
					pfx := ""
					if allowService {
						pfx = "un"
					}
					logrus.Infof("simpleNetworkService: %vblock service (allowService=%v)", pfx, allowService)

					sns.simpleNetworkServiceBlocked = !allowService
					// When service is blocked it implies that the southbound NSE gets also removed.
					// Removal of the NSE from registry prompts the NSC side to close the related NSM
					// connections making the associated interfaces unusable. However unfortunately
					// NSM is not able to properly close a connection associated with a "disappeared" NSE
					// (so NSM interfaces remain as well).
					//
					// Thus in SimpleNetworkService we must prohibit processing of new Targets and
					// creation of new southbound NSE interfaces while NSE removal takes effect on
					// NSC side.
					// Moreover the known Targets and thus the associated routing must be force removed.
					// That's because once the "block" is lifted, the southbound NSE should be advertised
					// again, resulting in new NSM Service Requests and thus interfaces for which the Target
					// routes must be readjusted.
					// Interference of old NSM interfaces must be avoided, thus their link state is changed
					// to down. (Hopefully once NSM finally decides to remove an old interface (e.g. due
					// to some timeout or whatever) this state change won't screw up things...)
					//
					// Note: Currently SimpleNetworkServiceClient/FullMeshNetworkServiceClient on the proxy side
					// will keep trying to establish an NSM connection forever, while also blocking NSE event
					// processing. So if the NSE disappeared in the meantime, it will go unnoticed by the proxy.
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
		if err != nil {
			logrus.Errorf("watchStreams err: %v", err)
		}
	}()
}

// InterfaceCreated -
func (sns *SimpleNetworkService) InterfaceCreated(intf networking.Iface) {
	logrus.Infof("SimpleNetworkService: InterfaceCreated: %v", intf)
	if sns.serviceBlocked() {
		// if service blocked, do not process new interface events (which
		// might appear until the block takes effect on NSC side)
		// instead disable them not to interfere after the block is lifted
		sns.disableInterface(intf)
		return
	}
	sns.interfaces.Store(intf.GetIndex(), intf)
}

// InterfaceDeleted -
func (sns *SimpleNetworkService) InterfaceDeleted(intf networking.Iface) {
	logrus.Infof("SimpleNetworkService: InterfaceDeleted: Intf %v", intf)
	sns.interfaces.Delete(intf.GetIndex())
}

func (sns *SimpleNetworkService) watchStreams(ctx context.Context) error {
	streamsToWatch := &nspAPI.Stream{
		Conduit: sns.Conduit,
	}
	watchStream, err := sns.ConfigurationManagerClient.WatchStream(ctx, streamsToWatch)
	if err != nil {
		return err
	}
	for {
		streamResponse, err := watchStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = sns.updateStreams(streamResponse.Streams)
		if err != nil {
			logrus.Errorf("updateStreams err: %v", err)
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
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
			}
		} else { // todo: check if need an update
			delete(remainingStreams, s.GetName())
		}
	}
	// remove remaining ones
	for streamName := range remainingStreams {
		err := sns.deleteStream(streamName)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}
	}
	// adjust stream service serving status (needs at least 1 stream)
	health.SetServingStatus(sns.ctx, health.StreamSvc, len(sns.streams) > 0)
	return errFinal
}

func (sns *SimpleNetworkService) addStream(strm *nspAPI.Stream) error {
	// verify if stream belongs to this conduit and trench
	_, exists := sns.streams[strm.GetName()]
	if exists {
		return errors.New("this stream already exists")
	}
	s, err := stream.New(
		strm,
		sns.targetRegistryClient,
		sns.ConfigurationManagerClient,
		M,
		N,
		sns.nfqueueIndex,
		sns.netUtils,
		sns.lbFactory,
	)
	if err != nil {
		return err
	}
	go func() {
		err := s.Start(sns.ctx)
		if err != nil {
			logrus.Errorf("stream start err: %v", err)
		}
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
	err := stream.Delete()
	delete(sns.streams, streamName)
	return err
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
	logrus.Infof("SimpleNetworkService: Evict Streams")
	for _, stream := range sns.streams {
		err := stream.Delete()
		if err != nil {
			logrus.Errorf("stream delete err: %v", err)
		}
	}
	sns.streams = make(map[string]types.Stream)
}

// disableInterfaces -
// Set interfaces down, so that they won't interface with future "Add Target"
// operation. Meaning old interfaces not yet removed by NSM must not get associated
// with routes inserted for Targets after the block is lifted.
func (sns *SimpleNetworkService) disableInterfaces() {
	logrus.Infof("SimpleNetworkService: Disable Interfaces")
	sns.interfaces.Range(func(key interface{}, value interface{}) bool {
		sns.disableInterface(value.(networking.Iface))
		sns.interfaces.Delete(key)
		return true
	})
}

// disableInterface -
// Set interface state down
func (sns *SimpleNetworkService) disableInterface(intf networking.Iface) {
	logrus.Debugf("SimpleNetworkService: Disable Intf %v", intf)
	la := netlink.NewLinkAttrs()
	la.Index = intf.GetIndex()
	err := netlink.LinkSetDown(&netlink.Dummy{LinkAttrs: la})
	if err != nil {
		logrus.Warnf("SimpleNetworkService: err Disable Intf (%v): %v", la.Index, err)
	}
}

// watchVips -
// Monitors VIP changes in Trench via NSP
func (sns *SimpleNetworkService) watchVips(ctx context.Context) {
	logrus.Infof("SimpleNetworkService: Watch VIPs")
	err := retry.Do(func() error {
		vipsToWatch := &nspAPI.Vip{
			Trench: sns.Conduit.GetTrench(),
		}
		watchVip, err := sns.ConfigurationManagerClient.WatchVip(ctx, vipsToWatch)
		if err != nil {
			return err
		}
		for {
			vipResponse, err := watchVip.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			err = sns.updateVips(vipResponse.GetVips())
			if err != nil {
				logrus.Errorf("updateVips err: %v", err)
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		logrus.Warnf("err watchVIPs: %v", err) // todo
	}
}

// updateVips -
// Sends list of VIPs to Netfilter Adaptor to adjust kerner based rules
func (sns *SimpleNetworkService) updateVips(vips []*nspAPI.Vip) error {
	logrus.Debugf("SimpleNetworkService: updateVips %v", vips)
	return sns.nfa.SetDestinationIPs(vips)
}
