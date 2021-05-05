package nsm

import (
	"context"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/api/pkg/api/registry"
	registryrefresh "github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// APIClient -
type APIClient struct {
	context                              context.Context
	GRPCClient                           *grpc.ClientConn
	config                               *Config
	x509source                           *workloadapi.X509Source
	NetworkServiceEndpointRegistryClient registry.NetworkServiceEndpointRegistryClient
	NetworkServiceRegistryClient         registry.NetworkServiceRegistryClient
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

func (apiClient *APIClient) setNetworkServiceEndpointRegistryClient() {
	apiClient.NetworkServiceEndpointRegistryClient = registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(apiClient.GRPCClient),
	)
}

func (apiClient *APIClient) setNetworkServiceRegistryClient() {
	apiClient.NetworkServiceRegistryClient = registry.NewNetworkServiceRegistryClient(apiClient.GRPCClient)
}

func (apiClient *APIClient) dial() {
	var err error

	apiClient.x509source = apiClient.getX509Source()

	connectCtx, cancel := context.WithTimeout(apiClient.context, apiClient.config.DialTimeout)
	defer cancel()

	apiClient.GRPCClient, err = grpc.DialContext(
		connectCtx,
		grpcutils.URLToTarget(&apiClient.config.ConnectTo),
		append(opentracing.WithTracingDial(),
			grpcfd.WithChainStreamInterceptor(),
			grpcfd.WithChainUnaryInterceptor(),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
				grpc.PerRPCCredentials(token.NewPerRPCCredentials(spiffejwt.TokenGeneratorFunc(apiClient.x509source, apiClient.config.MaxTokenLifetime))),
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

// NewAPIClient -
func NewAPIClient(ctx context.Context, config *Config) *APIClient {
	apiClient := &APIClient{
		context: ctx,
		config:  config,
	}

	apiClient.dial()
	apiClient.setNetworkServiceEndpointRegistryClient()
	apiClient.setNetworkServiceRegistryClient()

	return apiClient
}
