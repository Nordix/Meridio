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
	"fmt"
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

// TODO:
// There's another endpoint package with almost the same functionality:
// pkg/nsm/endpoint/endpoint.go
// Try to get rid of one of the two.

// Endpoint -
type Endpoint struct {
	config   *Config
	listenOn *url.URL
	tmpDir   string // used to generate unix socket path

	networkServiceRegistryClient         registryapi.NetworkServiceRegistryClient
	networkServiceEndpointRegistryClient registryapi.NetworkServiceEndpointRegistryClient

	nse    *registryapi.NetworkServiceEndpoint
	logger logr.Logger
	cancel context.CancelFunc // so that the gRPC server could be closed
}

// startWithoutRegister -
// Starts NSM endpoint along with the gRPC server according to the configuration.
// The NSM endpoint is not registered yet in NSM.
func (e *Endpoint) startWithoutRegister(ctx context.Context, additionalFunctionality ...networkservice.NetworkServiceServer) error {
	source, err := e.getSource(ctx)
	if err != nil {
		return err
	}

	responderEndpoint := endpoint.NewServer(ctx,
		spiffejwt.TokenGeneratorFunc(source, e.config.MaxTokenLifetime),
		endpoint.WithName(e.config.Name),
		endpoint.WithAuthorizeServer(authorize.NewServer()),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...))

	options := append(
		tracing.WithTracing(),
		grpc.Creds(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)
	server := grpc.NewServer(options...)
	responderEndpoint.Register(server)

	e.tmpDir, err = os.MkdirTemp("", e.config.Name)
	if err != nil {
		return errors.Wrap(err, "error creating tmpDir")
	}
	e.listenOn = &(url.URL{Scheme: "unix", Path: filepath.Join(e.tmpDir, "listen.on")})
	srvErrCh := grpcutils.ListenAndServe(ctx, e.listenOn, server) // note: also stops the server if the context is closed
	go e.errorHandler(srvErrCh)
	e.logger.Info("startWithoutRegister")

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

func (e *Endpoint) errorHandler(errCh <-chan error) {
	err := <-errCh
	e.logger.Error(err, "endpoint server errorHandler")
}

func (e *Endpoint) getSource(ctx context.Context) (*workloadapi.X509Source, error) {
	// retrieving svid, check spire agent logs if this is the last line you see
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting x509 source")
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		return nil, errors.Wrap(err, "error getting x509 svid")
	}
	e.logger.Info("getSource", "sVID", svid.ID)
	return source, nil
}

func (e *Endpoint) register(ctx context.Context) error {
	if e.nse == nil {
		return fmt.Errorf("registry api endpoint missing")
	}

	networkService, err := e.networkServiceRegistryClient.Register(ctx, &registryapi.NetworkService{
		Name:    e.config.ServiceName,
		Payload: payload.Ethernet,
	})
	if err != nil {
		return err
	}
	e.logger.Info("register", "networkService", networkService)

	e.nse.ExpirationTime = nil
	nse, err := e.networkServiceEndpointRegistryClient.Register(ctx, e.nse)
	if err != nil {
		return err
	}
	e.logger.Info("register", "nse", nse)

	return nil
}

// note: calling unregister without prior call to register
// will most likely lead to an error being returned by NSM
func (e *Endpoint) unregister(ctx context.Context) error {
	if e.nse == nil {
		return nil
	}

	e.nse.ExpirationTime = &timestamppb.Timestamp{
		Seconds: -1,
	}

	e.logger.Info("unregister", "nse", e.nse)
	_, err := e.networkServiceEndpointRegistryClient.Unregister(ctx, e.nse)
	return err
}

// Delete -
func (e *Endpoint) Delete(ctx context.Context) {
	logger := e.logger.WithValues("func", "Delete")
	logger.Info("Called")

	defer func() {
		if e.cancel != nil {
			e.cancel()
		}
	}()

	if e.nse != nil {
		if err := e.unregister(ctx); err != nil {
			logger.Error(err, "unregister")
		}
	}

	if e.tmpDir != "" {
		_ = os.Remove(e.tmpDir)
	}
}

// Register -
// Registers Network Service and Network Service Endpoint in NSM.
func (e *Endpoint) Register(ctx context.Context) error {
	return e.register(ctx)
}

// Unregister -
// Unregisters Network Service Endpoint in NSM.
func (e *Endpoint) Unregister(ctx context.Context) error {
	return e.unregister(ctx)
}

// GetUrl -
// Gets URL of the server
func (e *Endpoint) GetUrl() string {
	return e.listenOn.String()
}

// NewEndpoint -
// Creates and starts NSM endpoint according to the configuration, which can be
// registered or unregistered in NSM using the respective Register/Unregister methods.
//
// Note: on teardown if endpoint is expected to explicitly unregister,
// then the context shall not be cancelled/closed before Delete() is called
func NewEndpoint(
	ctx context.Context,
	config *Config,
	networkServiceRegistryClient registryapi.NetworkServiceRegistryClient,
	networkServiceEndpointRegistryClient registryapi.NetworkServiceEndpointRegistryClient,
	additionalFunctionality ...networkservice.NetworkServiceServer) (*Endpoint, error) {

	serverCtx, serverCancel := context.WithCancel(ctx)
	endpoint := &Endpoint{
		config:                               config,
		networkServiceRegistryClient:         networkServiceRegistryClient,
		networkServiceEndpointRegistryClient: networkServiceEndpointRegistryClient,
		logger:                               log.FromContextOrGlobal(ctx).WithValues("class", "Endpoint", "instance", config.Name),
		cancel:                               serverCancel,
	}

	err := endpoint.startWithoutRegister(serverCtx, additionalFunctionality...)
	if err != nil {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(time.Second))
		defer cancel()
		endpoint.Delete(ctx)
		return nil, err
	}

	return endpoint, nil
}
