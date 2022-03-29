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

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	registryrefresh "github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/sirupsen/logrus"
)

type FullMeshNetworkServiceClient struct {
	networkServiceClients                map[string]NetworkServiceClient
	client                               networkservice.NetworkServiceClient
	config                               *Config
	ctx                                  context.Context
	baseRequest                          *networkservice.NetworkServiceRequest
	networkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClient
	networkServiceDiscoveryStream        registry.NetworkServiceEndpointRegistry_FindClient
	mu                                   sync.Mutex
	serviceClosed                        bool
}

// Request -
// Blocks on listening for NSE add/delete events
func (fmnsc *FullMeshNetworkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !fmnsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}
	fmnsc.baseRequest = request
	query := fmnsc.prepareQuery()
	logrus.Debugf("Full Mesh Client: Request: %v", query)

	err := retry.Do(func() error {
		var err error
		fmnsc.networkServiceDiscoveryStream, err = fmnsc.networkServiceEndpointRegistryClient.Find(fmnsc.ctx, query)
		if err != nil {
			logrus.Debugf("Full Mesh Client: Find err: %v", err)
			return err
		}
		return fmnsc.recv()
	}, retry.WithContext(fmnsc.ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		logrus.Errorf("Full Mesh Client: err: %v", err)
	}

	return err
}

// Close -
// Note: adding further clients once closed must be avoided
func (fmnsc *FullMeshNetworkServiceClient) Close() error {
	logrus.Infof("Full Mesh Client: Close")
	fmnsc.mu.Lock()
	fmnsc.serviceClosed = true
	fmnsc.mu.Unlock()

	for networkServiceEndpointName := range fmnsc.networkServiceClients {
		logrus.Debugf("Full Mesh Client: Close %v", networkServiceEndpointName)
		fmnsc.deleteNetworkServiceClient(networkServiceEndpointName)
	}
	return nil
}

func (fmnsc *FullMeshNetworkServiceClient) recv() error {
	for {
		resp, err := fmnsc.networkServiceDiscoveryStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Debugf("Full Mesh Client: Recv err: %v", err)
			return err
		}
		networkServiceEndpoint := resp.NetworkServiceEndpoint
		if !expirationTimeIsNull(networkServiceEndpoint.ExpirationTime) && !resp.Deleted {
			fmnsc.addNetworkServiceClient(networkServiceEndpoint.Name)
		} else {
			logrus.Infof("Full Mesh Client: endpoint deleted or expired: %v", resp)
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
	// UUID part at the start of the conn id will be used by NSM to generate the interface name (we want it to be unique)
	request.Connection.Id = fmt.Sprintf("%s-%s-%s-%s", uuid.New().String(), fmnsc.config.Name, request.Connection.NetworkService, request.Connection.NetworkServiceEndpointName)
	logrus.Infof("Full Mesh Client: Add endpoint %v (service=%v, id=%v)", networkServiceEndpointName, request.Connection.NetworkService, request.Connection.Id)

	// Request would try forever, but what if the NetworkServiceEndpoint is removed in the meantime?
	// The recv() method must not be blocked by a pending Request that might not ever succeed.
	// Also, on NSE removal networkServiceClient must be capable of cancelling a pending request,
	// or closing the established connection.
	fmnsc.networkServiceClients[networkServiceEndpointName] = networkServiceClient
	go func() {
		err := networkServiceClient.Request(request)
		if err != nil {
			logrus.Errorf("Full Mesh Client: addNetworkServiceClient %v err: %v", networkServiceEndpointName, err)
		}
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
	logrus.Infof("Full Mesh Client: Delete endpoint %v (service: %v)", networkServiceEndpointName, fmnsc.baseRequest.Connection.NetworkService)
	err := networkServiceClient.Close()
	delete(fmnsc.networkServiceClients, networkServiceEndpointName)
	if err != nil {
		logrus.Errorf("Full Mesh Client: deleteNetworkServiceClient err: %v", err)
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
	}

	fullMeshNetworkServiceClient.networkServiceEndpointRegistryClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(ctx),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(config.APIClient.GRPCClient),
	)

	return fullMeshNetworkServiceClient
}
