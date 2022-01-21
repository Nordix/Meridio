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

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	registryrefresh "github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
)

type FullMeshNetworkServiceClient struct {
	networkServiceClient                 networkservice.NetworkServiceClient
	networkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClient
	baseRequest                          *networkservice.NetworkServiceRequest
	networkServiceDiscoveryStream        registry.NetworkServiceEndpointRegistry_FindClient
	config                               *Config
	mu                                   sync.Mutex
	networkServiceClients                map[string]*SimpleNetworkServiceClient
	ctx                                  context.Context
}

// Request -
func (fmnsc *FullMeshNetworkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !fmnsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}
	fmnsc.baseRequest = request
	query := fmnsc.prepareQuery()
	logrus.Debugf("Full Mesh: Request: %v", query)
	var err error
	// TODO: Context
	fmnsc.networkServiceDiscoveryStream, err = fmnsc.networkServiceEndpointRegistryClient.Find(fmnsc.ctx, query)
	if err != nil {
		logrus.Debugf("Full Mesh: Find err: %v", err)
		return err
	}
	return fmnsc.recv()
}

// Close -
func (fmnsc *FullMeshNetworkServiceClient) Close() error {
	for networkServiceEndpointName := range fmnsc.networkServiceClients {
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
			logrus.Debugf("Full Mesh: Recv err: %v", err)
			return err
		}
		networkServiceEndpoint := resp.NetworkServiceEndpoint
		if !expirationTimeIsNull(networkServiceEndpoint.ExpirationTime) && !resp.Deleted {
			fmnsc.addNetworkServiceClient(networkServiceEndpoint.Name)
		} else {
			logrus.Infof("Full Mesh: endpoint deleted or expired: %v", resp)
			fmnsc.deleteNetworkServiceClient(networkServiceEndpoint.Name)
		}
	}
	return nil
}

func (fmnsc *FullMeshNetworkServiceClient) addNetworkServiceClient(networkServiceEndpointName string) {
	fmnsc.mu.Lock()
	defer fmnsc.mu.Unlock()
	if fmnsc.networkServiceClientExists(networkServiceEndpointName) {
		return
	}
	networkServiceClient := &SimpleNetworkServiceClient{
		networkServiceClient: fmnsc.networkServiceClient,
		config:               fmnsc.config,
		ctx:                  fmnsc.ctx,
	}
	request := copyRequest(fmnsc.baseRequest)
	request.Connection.NetworkServiceEndpointName = networkServiceEndpointName
	request.Connection.Id = fmt.Sprintf("%s-%s-%s", fmnsc.config.Name, request.Connection.NetworkService, request.Connection.NetworkServiceEndpointName)
	// UUID part at the start of the conn id will be used by NSM to generate the interface name (we want it to be unique)
	request.Connection.Id = fmt.Sprintf("%s-%s-%s-%s", uuid.New().String(), fmnsc.config.Name, request.Connection.NetworkService, request.Connection.NetworkServiceEndpointName)
	logrus.Infof("Full Mesh Client (%v - %v): event add: %v", request.Connection.Id, request.Connection.NetworkService, networkServiceEndpointName)
	// TODO: Request tries forever, but what if the NSE is removed in the meantime?
	// The recv will be blocked on the Request as well... Should be refactored. (Are client components thread safe to opt for async requests?)

	err := networkServiceClient.Request(request)
	fmnsc.networkServiceClients[networkServiceEndpointName] = networkServiceClient
	if err != nil {
		logrus.Errorf("Full Mesh: addNetworkServiceClient err: %v", err)
	}
}

func (fmnsc *FullMeshNetworkServiceClient) deleteNetworkServiceClient(networkServiceEndpointName string) {
	fmnsc.mu.Lock()
	defer fmnsc.mu.Unlock()
	networkServiceClient, exists := fmnsc.networkServiceClients[networkServiceEndpointName]
	if !exists {
		return
	}
	logrus.Infof("Full Mesh Client (%v): event delete: %v", fmnsc.baseRequest.Connection.NetworkService, networkServiceEndpointName)
	err := networkServiceClient.Close()
	delete(fmnsc.networkServiceClients, networkServiceEndpointName)
	if err != nil {
		logrus.Errorf("Full Mesh: deleteNetworkServiceClient err: %v", err)
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
// Creates FullMeshNetworkServiceClient relying on NSM's client.NewClient API
func NewFullMeshNetworkServiceClient(ctx context.Context, config *Config, nsmAPIClient *nsm.APIClient, additionalFunctionality ...networkservice.NetworkServiceClient) NetworkServiceClient {
	fullMeshNetworkServiceClient := &FullMeshNetworkServiceClient{
		config:                config,
		networkServiceClient:  newClient(ctx, config.Name, nsmAPIClient, additionalFunctionality...),
		networkServiceClients: make(map[string]*SimpleNetworkServiceClient),
		ctx:                   ctx,
	}

	fullMeshNetworkServiceClient.networkServiceEndpointRegistryClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(ctx),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(nsmAPIClient.GRPCClient),
	)

	return fullMeshNetworkServiceClient
}
