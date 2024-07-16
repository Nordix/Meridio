/*
Copyright (c) 2021-2023 Nordix Foundation

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

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	registryrefresh "github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FullMeshNetworkServiceClient struct {
	networkServiceClients                map[string]NetworkServiceClient
	client                               networkservice.NetworkServiceClient
	config                               *Config
	ctx                                  context.Context
	baseRequest                          *networkservice.NetworkServiceRequest
	networkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClient
	mu                                   sync.Mutex
	serviceClosed                        bool
	logger                               logr.Logger
}

// Request -
// Blocks on listening for NSE add/delete events
// TODO: That NSM Find client seems unreliable; reports lost/new NSEs with delay.
// When scaled-in all LBs left the system as was, then scaled back LBs, old proxies were not notified about new LB NSEs.
// While a new proxy (e.g. after deleting an old POD) got the notification (NSM 1.11.1). Check how this is supposed to work!!!
func (fmnsc *FullMeshNetworkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !fmnsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}
	logger := fmnsc.logger.WithValues("func", "Request")
	fmnsc.baseRequest = request
	query := fmnsc.prepareQuery()
	logger.V(1).Info("Start network service endpoint discovery", "query", query)
	defer logger.Info("Stopped network service endpoint discovery")

	err := retry.Do(func() error {
		var err error
		networkServiceDiscoveryStream, err := fmnsc.networkServiceEndpointRegistryClient.Find(fmnsc.ctx, query)
		if err != nil {
			if status.Code(err) != codes.Canceled {
				fmnsc.logger.V(1).Info("Failed to create network service endpoint registry find client", "error", err)
			}
			return fmt.Errorf("failed to create network service endpoint registry find client: %w", err)
		}
		return fmnsc.recv(networkServiceDiscoveryStream)
	}, retry.WithContext(fmnsc.ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())

	if err != nil {
		return fmt.Errorf("network service endpoint event processing error: %w", err)
	}

	return nil
}

// Close -
// Note: adding further clients once closed must be avoided
// TODO: improve code to support parallel closing of connections (seems to require a lot of redesign)
func (fmnsc *FullMeshNetworkServiceClient) Close() error {
	logger := fmnsc.logger.WithValues("func", "Close")
	fmnsc.mu.Lock()
	fmnsc.serviceClosed = true
	fmnsc.mu.Unlock()

	logger.Info("Close and delete discovered network service endpoints", "num", len(fmnsc.networkServiceClients))
	for networkServiceEndpointName := range fmnsc.networkServiceClients {
		fmnsc.deleteNetworkServiceClient(networkServiceEndpointName)
	}
	return nil
}

func (fmnsc *FullMeshNetworkServiceClient) recv(networkServiceDiscoveryStream registry.NetworkServiceEndpointRegistry_FindClient) error {
	for {
		resp, err := networkServiceDiscoveryStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if status.Code(err) != codes.Canceled {
				fmnsc.logger.V(1).Info("Recv", "error", err)
			}
			return fmt.Errorf("network service endpoint registry find client receive error: %w", err)
		}
		networkServiceEndpoint := resp.NetworkServiceEndpoint
		if !expirationTimeIsNull(networkServiceEndpoint.ExpirationTime) && !resp.Deleted {
			fmnsc.addNetworkServiceClient(networkServiceEndpoint.Name)
		} else {
			fmnsc.logger.Info("Network service endpoint deleted or expired",
				"name", networkServiceEndpoint.Name, "resp", resp)
			fmnsc.deleteNetworkServiceClient(networkServiceEndpoint.Name)
		}
	}
	return nil
}

// addNetworkServiceClient -
// Adds new client and requests connection to the Network Service Endpoint in non-blocking manner
func (fmnsc *FullMeshNetworkServiceClient) addNetworkServiceClient(networkServiceEndpointName string) {
	fmnsc.mu.Lock()
	defer fmnsc.mu.Unlock()

	if fmnsc.serviceClosed || fmnsc.networkServiceClientExists(networkServiceEndpointName) {
		return
	}
	networkServiceClient := NewSimpleNetworkServiceClient(fmnsc.ctx, fmnsc.config, fmnsc.client)
	request := fmnsc.baseRequest.Clone()
	request.Connection.NetworkServiceEndpointName = networkServiceEndpointName
	// UUID part at the start of the conn id will be used by NSM to generate
	// the interface name (we want it to be unique).
	// The random part also secures that there should be no connections with
	// the same connection id maintened by NSMgr (for example after a possible
	// Proxy crash). Hence, no need for a NSM connection monitor to attempt
	// connection recovery when trying to connect the first time.
	// TODO: But the random part also implies, that after a crash the old IPs
	// will be be leaked.
	// TODO: Consider trying to recover a connection towards the new NSE via
	// NSM Monitor Connection, before generating an ID for a new connection.
	// That's because in case of an instable system NSEs might be reported
	// unaivalble and then shortly available again. NSM Close() related the
	// unavailable report might fail, thus leaving interfaces behind whose IPs
	// could get freed up before the old interfaces would disappear. Luckily,
	// the old interfaces should be removed from the bridge, thus hopefully not
	// causing problem just confusion.
	request.Connection.Id = fmt.Sprintf("%s-%s-%s-%s", uuid.New().String(),
		fmnsc.config.Name,
		request.Connection.NetworkService,
		request.Connection.NetworkServiceEndpointName,
	)
	logger := fmnsc.logger.WithValues("func", "addNetworkServiceClient",
		"name", networkServiceEndpointName,
		"service", request.Connection.NetworkService,
		"id", request.Connection.Id,
	)
	logger.Info("Add network service endpoint")

	// Request would try forever, but what if the NetworkServiceEndpoint is removed in the meantime?
	// The recv() method must not be blocked by a pending Request that might not ever succeed.
	// Also, on NSE removal networkServiceClient must be capable of cancelling a pending request,
	// or closing the established connection.
	fmnsc.networkServiceClients[networkServiceEndpointName] = networkServiceClient
	go func() {
		if err := networkServiceClient.Request(request); err != nil {
			logger.Error(err, "Failed to connect network service endpoint")
			return
		}
		logger.Info("Connected network service endpoint")
	}()
}

// deleteNetworkServiceClient -
// Deletes client and closes connection towards Network Service Endpoint
func (fmnsc *FullMeshNetworkServiceClient) deleteNetworkServiceClient(networkServiceEndpointName string) {
	fmnsc.mu.Lock()
	defer fmnsc.mu.Unlock()

	networkServiceClient, exists := fmnsc.networkServiceClients[networkServiceEndpointName]
	if !exists {
		return
	}
	logger := fmnsc.logger.WithValues("func", "deleteNetworkServiceClient",
		"name", networkServiceEndpointName,
		"service", fmnsc.baseRequest.Connection.NetworkService,
	)
	logger.Info("Delete network service endpoint")
	err := networkServiceClient.Close()
	delete(fmnsc.networkServiceClients, networkServiceEndpointName)
	if err != nil {
		logger.Error(err, "Failed to close network service endpoint")
	} else {
		logger.Info("Closed network service endpoint")
	}
}

func (fmnsc *FullMeshNetworkServiceClient) networkServiceClientExists(networkServiceEndpointName string) bool {
	_, ok := fmnsc.networkServiceClients[networkServiceEndpointName]
	return ok
}

func (fmnsc *FullMeshNetworkServiceClient) requestIsValid(request *networkservice.NetworkServiceRequest) bool {
	if request == nil {
		return false
	}
	if request.GetMechanismPreferences() == nil || len(request.GetMechanismPreferences()) == 0 {
		return false
	}
	if request.GetConnection() == nil || request.GetConnection().NetworkService == "" {
		return false
	}
	return true
}

func (fmnsc *FullMeshNetworkServiceClient) prepareQuery() *registry.NetworkServiceEndpointQuery {
	networkServiceEndpoint := &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{fmnsc.baseRequest.Connection.NetworkService},
	}
	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: networkServiceEndpoint,
		Watch:                  true,
	}
	return query
}

// NewFullMeshNetworkServiceClient -
// Creates FullMeshNetworkServiceClient that upon invoking Request blocks and starts
// monitoring Network Service Endpoints belonging to the Network Service of the request.
// Connects to each new Network Service Endpoint, and closes connection when a known
// endpoint disappears.
func NewFullMeshNetworkServiceClient(ctx context.Context, config *Config, additionalFunctionality ...networkservice.NetworkServiceClient) NetworkServiceClient {
	// create base client relying on NSM's client.NewClient API
	client := newClient(ctx, config.Name, config.APIClient, additionalFunctionality...)

	fullMeshNetworkServiceClient := &FullMeshNetworkServiceClient{
		networkServiceClients: make(map[string]NetworkServiceClient),
		client:                client,
		ctx:                   ctx,
		config:                config,
		logger:                log.FromContextOrGlobal(ctx).WithValues("class", "FullMeshNetworkServiceClient"),
	}

	fullMeshNetworkServiceClient.networkServiceEndpointRegistryClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(ctx),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(config.APIClient.GRPCClient),
	)

	return fullMeshNetworkServiceClient
}
