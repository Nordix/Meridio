package nsm

import (
	"context"

	"github.com/edwarnicke/grpcfd"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/nsurl"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	nsc "github.com/nordix/meridio/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type APIClient struct {
	context                       context.Context
	grpcClient                    *grpc.ClientConn
	networkServiceClient          networkservice.NetworkServiceClient
	networkServiceDiscoveryClient registry.NetworkServiceEndpointRegistryClient
	config                        *nsc.Config
	x509source                    *workloadapi.X509Source
}

// Find -
func (apiClient *APIClient) Find(networkServiceEndpointQuery *registry.NetworkServiceEndpointQuery) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return apiClient.networkServiceDiscoveryClient.Find(apiClient.context, networkServiceEndpointQuery)
}

// Request -
func (apiClient *APIClient) Request(networkServiceRequest *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return apiClient.networkServiceClient.Request(context.Background(), networkServiceRequest)
}

// Close -
func (apiClient *APIClient) Close(connection *networkservice.Connection) (*empty.Empty, error) {
	return apiClient.networkServiceClient.Close(context.Background(), connection)
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

func (apiClient *APIClient) setNetworkServiceClient() {
	tokenClient := token.NewClient()
	sendfdClient := sendfd.NewClient()

	apiClient.networkServiceClient = client.NewClient(
		apiClient.context,
		apiClient.config.Name,
		nil,
		spiffejwt.TokenGeneratorFunc(apiClient.x509source, apiClient.config.MaxTokenLifetime),
		apiClient.grpcClient,
		append(
			apiClient.getAdditionalFunctionality(),
			tokenClient,
			sendfdClient,
		)...,
	)
}

func (apiClient *APIClient) setNetworkServiceDiscoveryClient() {
	apiClient.networkServiceDiscoveryClient = registry.NewNetworkServiceEndpointRegistryClient(apiClient.grpcClient)
}

func (apiClient *APIClient) getAdditionalFunctionality() []networkservice.NetworkServiceClient {
	var clients []networkservice.NetworkServiceClient

	u := (*nsurl.NSURL)(&apiClient.config.NetworkServices[0])

	mech := u.Mechanism()

	switch mech.Type {
	case kernelmech.MECHANISM:
		clients = append(clients, kernel.NewClient())
	case vfiomech.MECHANISM:
		clients = append(clients, vfio.NewClient())
	}

	return clients
}

func (apiClient *APIClient) dial() {
	var err error

	apiClient.x509source = apiClient.getX509Source()

	connectCtx, cancel := context.WithTimeout(apiClient.context, apiClient.config.DialTimeout)
	defer cancel()

	apiClient.grpcClient, err = grpc.DialContext(
		connectCtx,
		grpcutils.URLToTarget(&apiClient.config.ConnectTo),
		append(opentracing.WithTracingDial(),
			grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
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
func NewAPIClient(ctx context.Context, config *nsc.Config) *APIClient {
	apiClient := &APIClient{
		context: ctx,
		config:  config,
	}

	apiClient.dial()
	apiClient.setNetworkServiceClient()
	apiClient.setNetworkServiceDiscoveryClient()

	return apiClient
}
