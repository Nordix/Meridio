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
	nscIndex                             int
	mu                                   sync.Mutex
	networkServiceClients                map[string]*SimpleNetworkServiceClient
}

// Request -
func (fmnsc *FullMeshNetworkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !fmnsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}
	fmnsc.baseRequest = request
	query := fmnsc.prepareQuery()
	var err error
	// TODO: Context
	fmnsc.networkServiceDiscoveryStream, err = fmnsc.networkServiceEndpointRegistryClient.Find(context.Background(), query)
	if err != nil {
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
		networkServiceEndpoint, err := fmnsc.networkServiceDiscoveryStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !expirationTimeIsNull(networkServiceEndpoint.ExpirationTime) {
			fmnsc.addNetworkServiceClient(networkServiceEndpoint.Name)
		} else {
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
	}
	request := copyRequest(fmnsc.baseRequest)
	request.Connection.NetworkServiceEndpointName = networkServiceEndpointName
	request.Connection.Id = fmt.Sprintf("%s-%s-%d", fmnsc.config.Name, request.Connection.NetworkService, fmnsc.nscIndex)
	fmnsc.nscIndex++
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
func NewFullMeshNetworkServiceClient(config *Config, nsmAPIClient *nsm.APIClient, additionalFunctionality ...networkservice.NetworkServiceClient) NetworkServiceClient {
	fullMeshNetworkServiceClient := &FullMeshNetworkServiceClient{
		config:                config,
		networkServiceClient:  newClient(context.Background(), config.Name, nsmAPIClient, additionalFunctionality...),
		networkServiceClients: make(map[string]*SimpleNetworkServiceClient),
		nscIndex:              0,
	}

	fullMeshNetworkServiceClient.networkServiceEndpointRegistryClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(context.Background()),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(nsmAPIClient.GRPCClient),
	)

	return fullMeshNetworkServiceClient
}
