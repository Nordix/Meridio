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

package endpoint

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/edwarnicke/grpcfd"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
	"github.com/nordix/meridio/pkg/log"
)

// Endpoint -
type Endpoint struct {
	context  context.Context
	config   *Config
	source   *workloadapi.X509Source
	listenOn *url.URL
	tmpDir   string

	networkServiceRegistryClient         registryapi.NetworkServiceRegistryClient
	networkServiceEndpointRegistryClient registryapi.NetworkServiceEndpointRegistryClient

	nse    *registryapi.NetworkServiceEndpoint
	logger logr.Logger
}

// Start -
func (e *Endpoint) Start(additionalFunctionality ...networkservice.NetworkServiceServer) error {
	err := e.StartWithoutRegister(additionalFunctionality...)
	if err != nil {
		return err
	}

	return e.register()
}

// Start -
func (e *Endpoint) StartWithoutRegister(additionalFunctionality ...networkservice.NetworkServiceServer) error {
	responderEndpoint := endpoint.NewServer(e.context,
		spiffejwt.TokenGeneratorFunc(e.source, e.config.MaxTokenLifetime),
		endpoint.WithName(e.config.Name),
		endpoint.WithAuthorizeServer(authorize.NewServer()),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...))

	options := append(
		tracing.WithTracing(),
		grpc.Creds(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSServerConfig(e.source, e.source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)
	server := grpc.NewServer(options...)
	responderEndpoint.Register(server)

	var err error
	e.tmpDir, err = os.MkdirTemp("", e.config.Name)
	if err != nil {
		return errors.Wrap(err, "error creating tmpDir")
	}
	e.listenOn = &(url.URL{Scheme: "unix", Path: filepath.Join(e.tmpDir, "listen.on")})
	srvErrCh := grpcutils.ListenAndServe(e.context, e.listenOn, server)
	go e.errorHandler(srvErrCh)
	e.logger.Info("StartWithoutRegister")

	e.nse = &registryapi.NetworkServiceEndpoint{
		Name:                e.config.Name,
		NetworkServiceNames: []string{e.config.ServiceName},
		NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
			e.config.ServiceName: {
				Labels: e.config.Labels,
			},
		},
		Url: e.listenOn.String(),
	}

	return nil
}

// ErrorHandler -
func (e *Endpoint) errorHandler(errCh <-chan error) {
	err := <-errCh
	e.logger.Error(err, "errorHandler")
}

// Delete -
func (e *Endpoint) Delete() {
	logger := e.logger.WithValues("func", "Delete")
	logger.Info("Called")
	ctx, cancel := context.WithTimeout(e.context, time.Duration(time.Second*3))
	defer cancel()

	if err := e.unregister(ctx); err != nil {
		logger.Error(err, "unregister")
	} else {
		logger.Info("unregistered")
	}
	_ = os.Remove(e.tmpDir)
}

func (e *Endpoint) setSource() error {
	// retrieving svid, check spire agent logs if this is the last line you see
	source, err := workloadapi.NewX509Source(e.context)
	if err != nil {
		return errors.Wrap(err, "Error getting x509 source")
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		return errors.Wrap(err, "Error getting x509 svid")
	}
	e.logger.Info("setSource", "sVID", svid.ID)
	e.source = source
	return nil
}

func (e *Endpoint) register() error {
	networkService, err := e.networkServiceRegistryClient.Register(e.context, &registryapi.NetworkService{
		Name:    e.config.ServiceName,
		Payload: payload.Ethernet,
	})
	e.logger.Info("register", "networkService", networkService)

	if err != nil {
		return errors.Wrap(err, "Error register network service")
	}

	e.nse.ExpirationTime = nil
	nse, err := e.networkServiceEndpointRegistryClient.Register(e.context, e.nse)
	e.logger.Info("register", "nse", nse)

	return err
}

func (e *Endpoint) unregister(ctx context.Context) error {
	e.nse.ExpirationTime = &timestamppb.Timestamp{
		Seconds: -1,
	}

	e.logger.Info("unregister", "nse", e.nse)
	_, err := e.networkServiceEndpointRegistryClient.Unregister(e.context, e.nse)
	if err != nil {
		e.logger.Error(err, "unregister")
	}

	return err
}

func (e *Endpoint) Announce() error {
	return e.register()
}

func (e *Endpoint) Denounce() error {
	return e.unregister(e.context)
}

func (e *Endpoint) GetUrl() string {
	return e.listenOn.String()
}

// NewEndpoint -
//
// Note: on teardown if endpoint is expected to explicitly unregister,
// then the context shall not be cancelled/closed before Delete() is called
func NewEndpoint(
	context context.Context,
	config *Config,
	networkServiceRegistryClient registryapi.NetworkServiceRegistryClient,
	networkServiceEndpointRegistryClient registryapi.NetworkServiceEndpointRegistryClient) (*Endpoint, error) {

	endpoint := &Endpoint{
		context:                              context,
		config:                               config,
		networkServiceRegistryClient:         networkServiceRegistryClient,
		networkServiceEndpointRegistryClient: networkServiceEndpointRegistryClient,
		logger:                               log.FromContextOrGlobal(context).WithValues("class", "Endpoint", "instrance", config.Name),
	}

	err := endpoint.setSource()
	if err != nil {
		return nil, errors.Wrap(err, "Error register network service")
	}

	return endpoint, nil
}
