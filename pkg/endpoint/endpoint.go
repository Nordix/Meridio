package endpoint

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/edwarnicke/grpcfd"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
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
}

// Start -
func (e *Endpoint) Start(additionalFunctionality ...networkservice.NetworkServiceServer) error {
	// create grpc server
	responderEndpoint := endpoint.NewServer(
		e.context,
		e.config.Name,
		authorize.NewServer(),
		spiffejwt.TokenGeneratorFunc(e.source, e.config.MaxTokenLifetime),
		additionalFunctionality...)

	options := append(
		opentracing.WithTracing(),
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
	e.tmpDir, err = ioutil.TempDir("", e.config.Name)
	if err != nil {
		return errors.Wrap(err, "error creating tmpDir")
	}
	e.listenOn = &(url.URL{Scheme: "unix", Path: filepath.Join(e.tmpDir, "listen.on")})
	srvErrCh := grpcutils.ListenAndServe(e.context, e.listenOn, server)
	go e.errorHandler(srvErrCh)
	logrus.Infof("grpc server started")

	return e.register()
}

// ErrorHandler -
func (e *Endpoint) errorHandler(errCh <-chan error) {
	err := <-errCh
	logrus.Error(err)
}

// Delete -
func (e *Endpoint) Delete() {
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
	logrus.Infof("SVID: %q", svid.ID)
	e.source = source
	return nil
}

func (e *Endpoint) register() error {
	_, err := e.networkServiceRegistryClient.Register(context.Background(), &registryapi.NetworkService{
		Name:    e.config.ServiceName,
		Payload: payload.IP,
	})

	if err != nil {
		return errors.Wrap(err, "Error register network service")
	}

	nse, err := e.networkServiceEndpointRegistryClient.Register(context.Background(), &registryapi.NetworkServiceEndpoint{
		Name:                e.config.Name,
		NetworkServiceNames: []string{e.config.ServiceName},
		NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
			e.config.ServiceName: {
				Labels: e.config.Labels,
			},
		},
		Url: e.listenOn.String(),
	})
	logrus.Infof("nse: %+v", nse)

	return err
}

// NewEndpoint -
func NewEndpoint(
	context context.Context,
	config *Config,
	networkServiceRegistryClient registryapi.NetworkServiceRegistryClient,
	networkServiceEndpointRegistryClient registryapi.NetworkServiceEndpointRegistryClient) *Endpoint {

	endpoint := &Endpoint{
		context:                              context,
		config:                               config,
		networkServiceRegistryClient:         networkServiceRegistryClient,
		networkServiceEndpointRegistryClient: networkServiceEndpointRegistryClient,
	}

	endpoint.setSource()

	return endpoint
}
