/*
Copyright (c) 2021-2023 Nordix Foundation
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

package trench

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/nordix/meridio/pkg/security/credentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
)

const grpcKeepaliveTime = 20 * time.Second

// Trench implements types.Trench (/pkg/ambassador/tap/types)
// Responsible for connection/disconnecting the conduits, and providing
// a NSP connection to the trench.
type Trench struct {
	Trench *ambassadorAPI.Trench
	// unique name (to be used in the NSM connection IDs)
	TargetName string
	// namespace of the current trench
	Namespace string
	// node the pod is running on
	NodeName                   string
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	TargetRegistryClient       nspAPI.TargetRegistryClient
	NetworkServiceClient       networkservice.NetworkServiceClient
	StreamRegistry             types.Registry
	NetUtils                   networking.Utils
	ConduitFactory             ConduitFactory
	Timeout                    time.Duration
	nspConn                    *grpc.ClientConn
	conduits                   []*conduitConnect
	mu                         sync.Mutex
	logger                     logr.Logger
}

// New is the constructor of Trench.
// The constructor will create a new conduit factory, connect to the
// NSP service (Configuration and Target registry).
func New(trench *ambassadorAPI.Trench,
	targetName string,
	namespace string,
	nodeName string,
	networkServiceClient networkservice.NetworkServiceClient,
	monitorConnectionClient networkservice.MonitorConnectionClient,
	streamRegistry types.Registry,
	nspServiceName string,
	nspServicePort int,
	nspEntryTimeout time.Duration,
	grpcMaxBackoff time.Duration,
	timeout time.Duration,
	netUtils networking.Utils) (*Trench, error) {

	logger := log.Logger.WithValues("class", "Trench", "trench", trench.GetName())
	logger.Info("Create trench")

	t := &Trench{
		TargetName:           targetName,
		Namespace:            namespace,
		NodeName:             nodeName,
		Trench:               trench,
		NetworkServiceClient: networkServiceClient,
		StreamRegistry:       streamRegistry,
		NetUtils:             netUtils,
		conduits:             []*conduitConnect{},
		Timeout:              timeout,
		logger:               logger,
	}

	var err error
	t.nspConn, err = t.connectNSPService(context.TODO(), nspServiceName, nspServicePort, grpcMaxBackoff)
	if err != nil {
		return nil, err
	}

	t.ConfigurationManagerClient = nspAPI.NewConfigurationManagerClient(t.nspConn)
	t.TargetRegistryClient = nspAPI.NewTargetRegistryClient(t.nspConn)
	t.ConduitFactory = newConduitFactoryImpl(t.TargetName,
		t.Namespace,
		t.NodeName,
		t.ConfigurationManagerClient,
		t.TargetRegistryClient,
		t.NetworkServiceClient,
		monitorConnectionClient,
		t.StreamRegistry,
		t.NetUtils,
		nspEntryTimeout)
	t.logger.Info("Created", "TrenchObject", t)
	return t, nil
}

func (t *Trench) Delete(ctx context.Context) error {
	t.logger.Info("Delete trench")
	t.mu.Lock()
	defer t.mu.Unlock()
	var errFinal error
	var err error
	// close streams
	streamsCtx, streamsCancel := context.WithTimeout(ctx, t.Timeout)
	err = t.closeStreams(streamsCtx)
	if err != nil {
		errFinal = fmt.Errorf("failure during close streams: %w", err) // todo
	}
	t.logger.Info("Streams closed", "error", err)
	streamsCancel()

	// disconnect conduits
	conduitsCtx, conduitsCancel := context.WithTimeout(ctx, t.Timeout)
	err = t.disconnectConduits(conduitsCtx)
	if err != nil {
		errFinal = fmt.Errorf("%w; failure during disconnect conduits: %w", errFinal, err) // todo
	}
	t.conduits = []*conduitConnect{}
	t.logger.Info("Conduits disconnected", "error", err)
	conduitsCancel()

	// disconnect trench related services (connection to NSP)
	err = t.nspConn.Close()
	if err != nil {
		errFinal = fmt.Errorf("%w; failure during nsp connection close: %w", errFinal, err) // todo
	}
	t.logger.Info("NSP connection closed", "error", err)
	t.ConfigurationManagerClient = nil
	t.TargetRegistryClient = nil
	t.nspConn = nil
	return errFinal
}

// AddConduit creates a conduit based on its factory and will connect it (in another goroutine)
func (t *Trench) AddConduit(ctx context.Context, cndt *ambassadorAPI.Conduit) (types.Conduit, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	c := t.getConduit(cndt)
	if c != nil {
		return c, nil
	}
	t.logger.Info("Add conduit", "conduit", cndt)
	c, err := t.ConduitFactory.New(cndt)
	if err != nil {
		return nil, fmt.Errorf("conduit create failed: %w", err)
	}
	cc := newConduitConnect(c, t.ConfigurationManagerClient, t.Timeout)
	go cc.connect()
	t.conduits = append(t.conduits, cc)
	return c, nil
}

// RemoveConduit disconnects and removes the conduit (if existing).
// TODO: If the conduit still has streams, they will not be removed from stream registry:
// 1. return an error
// 2. Remove them
func (t *Trench) RemoveConduit(ctx context.Context, cndt *ambassadorAPI.Conduit) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	index := t.getConduitIndex(cndt)
	if index < 0 {
		return nil
	}
	t.logger.Info("Remove conduit", "conduit", cndt)
	c := t.conduits[index]
	err := c.disconnect(ctx)
	t.conduits = append(t.conduits[:index], t.conduits[index+1:]...)
	return err
}

// GetConduits returns all conduits previously added to this trench.
func (t *Trench) GetConduits() []types.Conduit {
	t.mu.Lock()
	defer t.mu.Unlock()
	conduits := []types.Conduit{}
	for _, conduit := range t.conduits {
		conduits = append(conduits, conduit.conduit)
	}
	return conduits
}

// GetConduit returns the conduit corresponding to the one in parameter if it exists.
func (t *Trench) GetConduit(conduit *ambassadorAPI.Conduit) types.Conduit {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.getConduit(conduit)
}

// Equals checks if the trench is equal to the one in parameter.
func (t *Trench) Equals(trench *ambassadorAPI.Trench) bool {
	return t.Trench.Equals(trench)
}

func (t *Trench) GetTrench() *ambassadorAPI.Trench {
	return t.Trench
}

func (t *Trench) connectNSPService(ctx context.Context,
	nspServiceName string,
	nspServicePort int,
	grpcMaxBackoff time.Duration) (*grpc.ClientConn, error) {
	service := nsp.GetService(nspServiceName, t.Trench.GetName(), t.Namespace, nspServicePort)
	t.logger.Info("Connect to NSP Service", "service", service)
	// Allow changing max backoff delay from gRPC default 120s to limit reconnect interval.
	// Thus, allow faster reconnect if NSP has been unavailable. Otherwise gRPC might
	// wait up to 2 minutes to attempt reconnect due to the default backoff algorithm.
	// (refer to: https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md)
	grpcBackoffCfg := backoff.DefaultConfig
	if grpcBackoffCfg.MaxDelay != grpcMaxBackoff {
		grpcBackoffCfg.MaxDelay = grpcMaxBackoff
	}
	cc, err := grpc.Dial(service,
		grpc.WithTransportCredentials(
			credentials.GetClient(ctx),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: grpcBackoffCfg,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			// if the NSP service is re-created, the TAPA will take around 15 minutes to re-connect to the NSP service without this setting.
			Time: grpcKeepaliveTime,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("NSP service dial failed: %w", err)
	}
	return cc, nil
}

func (t *Trench) closeStreams(ctx context.Context) error {
	streams := []*ambassadorAPI.Stream{}
	for _, c := range t.conduits {
		streams = append(streams, c.conduit.GetStreams()...)
	}
	var wg sync.WaitGroup
	wg.Add(len(streams))
	var errFinal error
	var mu sync.Mutex
	for _, c := range t.conduits {
		for _, s := range c.conduit.GetStreams() {
			conduit := c
			go func(stream *ambassadorAPI.Stream) {
				defer wg.Done()
				err := conduit.conduit.RemoveStream(ctx, stream) // todo: retry
				if err != nil {
					mu.Lock()
					errFinal = fmt.Errorf("%w; %w", errFinal, fmt.Errorf("failure during removing stream %v from conduit %v: %w",
						stream.GetName(), conduit.conduit.GetConduit().Name, err)) // todo
					mu.Unlock()
				}
			}(s)
		}
	}
	wg.Wait()
	return errFinal
}

func (t *Trench) disconnectConduits(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(len(t.conduits))
	var errFinal error
	var mu sync.Mutex
	for _, c := range t.conduits {
		go func(conduit *conduitConnect) {
			defer wg.Done()
			err := conduit.disconnect(ctx) // todo: retry
			if err != nil {
				mu.Lock()
				errFinal = fmt.Errorf("%w; %w", errFinal, fmt.Errorf("failure during disconnect conduit %v: %w",
					conduit.conduit.GetConduit().Name, err)) // todo
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()
	return errFinal
}

func (t *Trench) getConduitIndex(cndt *ambassadorAPI.Conduit) int {
	for i, c := range t.conduits {
		equal := c.conduit.Equals(cndt)
		if equal {
			return i
		}
	}
	return -1
}

func (t *Trench) getConduit(cndt *ambassadorAPI.Conduit) types.Conduit {
	index := t.getConduitIndex(cndt)
	if index < 0 {
		return nil
	}
	return t.conduits[index].conduit
}
