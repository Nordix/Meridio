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

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/go-logr/logr"
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
	networkServiceEndpoints              map[string]*registry.NetworkServiceEndpoint // known endpoints
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
// Note: NSM Find client is unreliable; might report NSE create/update with delay and
// might miss to report NSE delete completely.
// TODO: When scaled-in all LBs left the system as was, then scaled back LBs, old proxies were not notified about new LB NSEs.
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
		// Note: Our NetworkServiceEndpointRegistry_FindClient recv will return every 15 seconds apparently. Otherwise,
		// it would return after 1 minute if k8s-registry is in use due to how the k8s client Watch object is created by NSM.
		// (refer to: https://github.com/networkservicemesh/sdk-k8s/blob/release/v1.13.2/pkg/registry/etcd/nse_server.go#L173)
		networkServiceDiscoveryStream, err := fmnsc.networkServiceEndpointRegistryClient.Find(fmnsc.ctx, query)
		if err != nil {
			if status.Code(err) != codes.Canceled {
				fmnsc.logger.V(1).Info("Failed to create network service endpoint registry find client", "error", err)
			}
			return fmt.Errorf("failed to create network service endpoint registry find client: %w", err)
		}
		err = fmnsc.recv(networkServiceDiscoveryStream)
		if err != nil {
			fmnsc.logger.Info("NetworkServiceEndpointRegistry_FindClient recv failed", "err", err)
		}

		// Because a new Find stream must be opened periodically, there can be gaps
		// when NSE removals could be missed. Therefore, whenever Find stream "returns",
		// check if any of the endpoints known to FullMeshNetworkServiceClient have
		// expired. And if so, remove them.
		fmnsc.checkEndpointExpiration()
		return err
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
			fmnsc.addNetworkServiceEndpoint(networkServiceEndpoint)
		} else {
			fmnsc.logger.Info("Network service endpoint deleted or expired",
				"name", networkServiceEndpoint.Name, "resp", resp)
			fmnsc.deleteNetworkServiceEndpoint(networkServiceEndpoint)
		}
	}
	return nil
}

// addNetworkServiceEndpoint adds or overwrites network service endpoint to keep track of
// its expiration time.
func (fmnsc *FullMeshNetworkServiceClient) addNetworkServiceEndpoint(endpoint *registry.NetworkServiceEndpoint) {
	fmnsc.networkServiceEndpoints[endpoint.Name] = endpoint
	fmnsc.addNetworkServiceClient(endpoint.Name)
}

func (fmnsc *FullMeshNetworkServiceClient) deleteNetworkServiceEndpoint(endpoint *registry.NetworkServiceEndpoint) {
	delete(fmnsc.networkServiceEndpoints, endpoint.Name)
	fmnsc.deleteNetworkServiceClient(endpoint.Name)
}

// checkEndpointExpiration checks the expiratiom time of the locally stored
// network service endpoints to determine if they are still valid.
// The logic aims to complement NSM NetworkServiceEndpointRegistry_FindClient
// based watch logic. So that endpoints for whom the delete notification
// happened to be missed via Find could be removed.
// (refer to: https://github.com/networkservicemesh/sdk-k8s/issues/512)
//
// Note: Removing endpoints based on expiration time shouldn't cause any problems,
// since heal should be informed right in time when sg happens to an LB endpoint.
// This periodic check merely stands to ensure the proxy won't keep spamming NSM
// "forever" with requests for which the endpoint is long gone.
func (fmnsc *FullMeshNetworkServiceClient) checkEndpointExpiration() {
	logger := fmnsc.logger.WithValues("func", "checkEndpointExpiration")
	for name, endpoint := range fmnsc.networkServiceEndpoints {
		if fmnsc.ctx.Err() != nil {
			// If context is closed FullMeshNetworkServiceClient.Close() is expected anyways to clen up...
			return
		}
		// Endpoint is considered expired if based on its ExpirationTime it's
		// been outdated for at least MaxTokenLifetime seconds. The reason why
		// an additional MaxTokenLifetime/2 seconds is used is because NSM might
		// not pass all updates in time via Find (refer to the NSM issue above).
		// In our case the delay can be 15 seconds as at the start of each Find
		// the list of available endpoints is returned anyways by NSM (v1.13.2).
		// So, the additional MaxTokenLifetime/2 delay is an overkill.
		if endpoint.ExpirationTime != nil &&
			endpoint.ExpirationTime.AsTime().Local().Add(fmnsc.config.MaxTokenLifetime/2).Before(time.Now()) {
			logger.Info("Network service endpoint expired", "name", name, "endpoint", endpoint)
			fmnsc.deleteNetworkServiceEndpoint(endpoint)
		}
	}

}

// addNetworkServiceClient -
// Adds new client and requests connection to the Network Service Endpoint in non-blocking manner
func (fmnsc *FullMeshNetworkServiceClient) addNetworkServiceClient(networkServiceEndpointName string) {
	fmnsc.mu.Lock()
	defer fmnsc.mu.Unlock()

	if fmnsc.serviceClosed || fmnsc.networkServiceClientExists(networkServiceEndpointName) {
		return
	}

	var monitoredConnections map[string]*networkservice.Connection
	networkServiceClient := NewSimpleNetworkServiceClient(fmnsc.ctx, fmnsc.config, fmnsc.client)
	request := fmnsc.baseRequest.Clone()
	request.Connection.NetworkServiceEndpointName = networkServiceEndpointName
	// Note: Starting with NSM v1.13.2 the NSM interface name does NOT depend
	// on the connection ID. Therefore, the random UUID part had been removed
	// from the ID. This allows the proxy to attempt recovery of any previous
	// connections after crash or temporary LB NSE unavailability (enabling
	// recovery of assocaited IPs as well). This can be achieved by using NSM's
	// connection monitor feature when connecting an LB NSE.
	id := fmt.Sprintf("%s-%s-%s",
		fmnsc.config.Name,
		request.Connection.NetworkService,
		request.Connection.NetworkServiceEndpointName,
	)
	request.Connection.Id = id
	logger := fmnsc.logger.WithValues("func", "addNetworkServiceClient",
		"name", networkServiceEndpointName,
		"service", request.Connection.NetworkService,
		"id", request.Connection.Id,
	)
	logger.Info("Add network service endpoint")

	// Check if NSM already tracks a connection with the same ID, if it does
	// then re-use the connection
	if fmnsc.config.MonitorConnectionClient != nil {
		monitorCli := fmnsc.config.MonitorConnectionClient
		stream, err := monitorCli.MonitorConnections(fmnsc.ctx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{
				{
					Id: id,
				},
			},
		})
		if err != nil {
			logger.Error(err, "Failed to create monitorConnectionClient")
		} else {
			event, err := stream.Recv()
			if err != nil {
				// probably running a really old NSM version, don't like this but let it continue anyways
				// XXX: I guess nsmgr crash/upgrade would cause EOF here, after which it would
				// make no sense trying again to fetch connections from an "empty" nsmgr...
				logger.Error(err, "error from monitorConnection stream")
			} else {
				monitoredConnections = event.Connections
			}
		}
	}

	// Update request based on recovered connection(s) if any
	for _, conn := range monitoredConnections {
		path := conn.GetPath()
		if path != nil && path.Index == 1 && path.PathSegments[0].Id == id && conn.Mechanism.Type == request.MechanismPreferences[0].Type {
			logger.Info("Recovered connection", "connection", conn)
			// TODO: consider merging any possible labels from baseRequest
			request.Connection = conn
			request.Connection.Path.Index = 0
			request.Connection.Id = id
			break
		}
	}

	// Check if recovered connection indicates issue with control plane,
	// if so request reselect. Otherwise, the connection request might fail
	// if an old path segment (e.g. forwarder) was replaced in the meantime.
	// (refer to https://github.com/networkservicemesh/cmd-nsc/pull/600)
	if request.GetConnection().State == networkservice.State_DOWN ||
		request.Connection.NetworkServiceEndpointName != networkServiceEndpointName {
		logger.Info("Request reselect for recovered connection")
		request.GetConnection().Mechanism = nil
		request.GetConnection().NetworkServiceEndpointName = networkServiceEndpointName // must be a valid reselect request because fullmeshtracker in case of heal with reconnect would do the same
		request.GetConnection().State = networkservice.State_RESELECT_REQUESTED
	}

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
		networkServiceClients:   make(map[string]NetworkServiceClient),
		networkServiceEndpoints: make(map[string]*registry.NetworkServiceEndpoint),
		client:                  client,
		ctx:                     ctx,
		config:                  config,
		logger:                  log.FromContextOrGlobal(ctx).WithValues("class", "FullMeshNetworkServiceClient"),
	}

	fullMeshNetworkServiceClient.networkServiceEndpointRegistryClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(ctx),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(config.APIClient.GRPCClient),
	)

	return fullMeshNetworkServiceClient
}
