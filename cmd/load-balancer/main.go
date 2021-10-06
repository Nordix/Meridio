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
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/health"
	linuxKernel "github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/loadbalancer/stream"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
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

	netUtils := &linuxKernel.KernelUtils{}

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

	conn, err := grpc.Dial(config.NSPService, grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Errorf("grpc.Dial err: %v", err)
	}
	targetRegistryClient := nspAPI.NewTargetRegistryClient(conn)
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(conn)
	conduit := &nspAPI.Conduit{
		Name: config.ConduitName,
		Trench: &nspAPI.Trench{
			Name: config.TrenchName,
		},
	}
	sns := NewSimpleNetworkService(ctx, targetRegistryClient, configurationManagerClient, conduit, netUtils)

	interfaceMonitor, err := netUtils.NewInterfaceMonitor()
	if err != nil {
		logrus.Fatalf("Error creating link monitor: %+v", err)
	}
	interfaceMonitorEndpoint := interfacemonitor.NewServer(interfaceMonitor, sns, netUtils)

	responderEndpoint := []networkservice.NetworkServiceServer{
		recvfd.NewServer(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		interfacename.NewServer("nse", &interfacename.RandomGenerator{}),
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
	nsmAPIClient := nsm.NewAPIClient(ctx, apiClientConfig)

	endpointConfig := &endpoint.Config{
		Name:             config.Name,
		ServiceName:      config.ServiceName,
		Labels:           make(map[string]string),
		MaxTokenLifetime: config.MaxTokenLifetime,
	}
	ep, err := endpoint.NewEndpoint(ctx, endpointConfig, nsmAPIClient.NetworkServiceRegistryClient, nsmAPIClient.NetworkServiceEndpointRegistryClient)
	if err != nil {
		logrus.Fatalf("unable to create a new nse %+v", err)
	}

	err = ep.StartWithoutRegister(responderEndpoint...)
	if err != nil {
		logrus.Fatalf("unable to start nse %+v", err)
	}

	defer ep.Delete()

	sns.Start()
	// monitor availibilty of frontends; if no feasible FE don't advertise NSE to proxies
	fns := NewFrontendNetworkService(targetRegistryClient, ep, NewServiceControlDispatcher(sns))
	fns.Start()

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
func NewSimpleNetworkService(ctx context.Context, targetRegistryClient nspAPI.TargetRegistryClient, configurationManagerClient nspAPI.ConfigurationManagerClient, conduit *nspAPI.Conduit, netUtils networking.Utils) *SimpleNetworkService {
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
	}
	return simpleNetworkService
}

// Start -
func (sns *SimpleNetworkService) Start() {
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
	var ctx context.Context
	ctx, sns.cancelStreamWatcher = context.WithCancel(sns.ctx)
	go func() {
		err := sns.watchStreams(ctx)
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
	return errFinal
}

func (sns *SimpleNetworkService) addStream(strm *nspAPI.Stream) error {
	// verify if stream belongs to this conduit and trench
	_, exists := sns.streams[strm.GetName()]
	if exists {
		return errors.New("this stream already exists")
	}
	s, err := stream.New(strm, sns.targetRegistryClient, sns.ConfigurationManagerClient, M, N, sns.nfqueueIndex, sns.netUtils)
	if err != nil {
		return err
	}
	go func() {
		err := s.Start(context.Background()) // todo
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
	if err != nil {
		return err
	}
	delete(sns.streams, streamName)
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
