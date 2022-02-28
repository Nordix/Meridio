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

package nsm

import (
	"context"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type NetworkServiceClient interface {
	networkservice.NetworkServiceClient
}

// APIClient -
type APIClient struct {
	context                              context.Context
	cancel                               context.CancelFunc
	GRPCClient                           *grpc.ClientConn
	Config                               *Config
	x509source                           *workloadapi.X509Source
	NetworkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClient
	NetworkServiceRegistryClient         registry.NetworkServiceRegistryClient
	GRPCDialOption                       []grpc.DialOption
}

func (apiClient *APIClient) getX509Source() *workloadapi.X509Source {
	source, err := workloadapi.NewX509Source(apiClient.context)
	if err != nil {
		logrus.Errorf("error getting x509 source: %v", err)
	}
	var svid *x509svid.SVID
	svid, err = source.GetX509SVID()
	if err != nil {
		logrus.Errorf("error getting x509 svid: %v", err)
	}
	logrus.Infof("sVID: %q", svid.ID)
	return source
}

func (apiClient *APIClient) GetClientOptions() []grpc.DialOption {
	return append(
		tracing.WithTracingDial(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSClientConfig(apiClient.x509source, apiClient.x509source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)
}

func (apiClient *APIClient) setNetworkServiceEndpointRegistryClient() {
	clientOptions := apiClient.GetClientOptions()
	apiClient.NetworkServiceEndpointRegistryClient = registryclient.NewNetworkServiceEndpointRegistryClient(
		apiClient.context,
		&apiClient.Config.ConnectTo,
		registryclient.WithDialOptions(clientOptions...),
		registryclient.WithNSEAdditionalFunctionality(
			registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		))
}

func (apiClient *APIClient) setNetworkServiceRegistryClient() {
	clientOptions := apiClient.GetClientOptions()
	apiClient.NetworkServiceRegistryClient = registryclient.NewNetworkServiceRegistryClient(
		apiClient.context,
		&apiClient.Config.ConnectTo,
		registryclient.WithDialOptions(clientOptions...))
}

func (apiClient *APIClient) dial() {
	var err error

	apiClient.x509source = apiClient.getX509Source()

	connectCtx, cancel := context.WithTimeout(apiClient.context, apiClient.Config.DialTimeout)
	defer cancel()

	apiClient.GRPCClient, err = grpc.DialContext(
		connectCtx,
		grpcutils.URLToTarget(&apiClient.Config.ConnectTo),
		append(tracing.WithTracingDial(),
			grpcfd.WithChainStreamInterceptor(),
			grpcfd.WithChainUnaryInterceptor(),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
				grpc.PerRPCCredentials(token.NewPerRPCCredentials(spiffejwt.TokenGeneratorFunc(apiClient.x509source, apiClient.Config.MaxTokenLifetime))),
			),
			grpc.WithTransportCredentials(
				grpcfd.TransportCredentials(
					credentials.NewTLS(
						tlsconfig.MTLSClientConfig(apiClient.x509source, apiClient.x509source, tlsconfig.AuthorizeAny()),
					),
				),
			))...,
	)
	if err != nil {
		logrus.Errorf("error dial Context: %v", err.Error())
	}
}

func (apiClient *APIClient) dialOptions() {
	apiClient.GRPCDialOption = append(tracing.WithTracingDial(),
		grpcfd.WithChainStreamInterceptor(),
		grpcfd.WithChainUnaryInterceptor(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(spiffejwt.TokenGeneratorFunc(apiClient.x509source, apiClient.Config.MaxTokenLifetime))),
		),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSClientConfig(apiClient.x509source, apiClient.x509source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)
}

// Delete -
// Cancels the context to tear down apiClient
func (apiClient *APIClient) Delete() {
	if apiClient.cancel != nil {
		logrus.Infof("apiClient: Delete")
		apiClient.cancel()
	}
}

// NewAPIClient -
func NewAPIClient(ctx context.Context, config *Config) *APIClient {
	ctx, cancel := context.WithCancel(ctx)
	apiClient := &APIClient{
		context:        ctx,
		cancel:         cancel,
		Config:         config,
		GRPCDialOption: []grpc.DialOption{},
	}

	apiClient.dial()
	apiClient.dialOptions()
	apiClient.setNetworkServiceEndpointRegistryClient()
	apiClient.setNetworkServiceRegistryClient()

	return apiClient
}
