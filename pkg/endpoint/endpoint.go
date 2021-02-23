package endpoint

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/edwarnicke/grpcfd"
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
	registryrefresh "github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
)

type Endpoint struct {
	Context  context.Context
	Config   *Config
	source   *workloadapi.X509Source
	listenOn *url.URL
	tmpDir   string
}

func (e *Endpoint) Start(additionalFunctionality ...networkservice.NetworkServiceServer) error {
	// create grpc server
	responderEndpoint := endpoint.NewServer(
		e.Context,
		e.Config.Name,
		authorize.NewServer(),
		spiffejwt.TokenGeneratorFunc(e.source, e.Config.MaxTokenLifetime),
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
	e.tmpDir, err = ioutil.TempDir("", e.Config.Name)
	if err != nil {
		logrus.Fatalf("error creating tmpDir %+v", err)
	}
	e.listenOn = &(url.URL{Scheme: "unix", Path: filepath.Join(e.tmpDir, "listen.on")})
	srvErrCh := grpcutils.ListenAndServe(e.Context, e.listenOn, server)
	go e.ErrorHandler(srvErrCh)
	log.FromContext(e.Context).Infof("grpc server started")

	return e.register()
}

func (e *Endpoint) ErrorHandler(errCh <-chan error) {
	err := <-errCh
	log.FromContext(e.Context).Error(err)
}

func (e *Endpoint) Delete() {
	_ = os.Remove(e.tmpDir)
}

func (e *Endpoint) setSource() {
	// retrieving svid, check spire agent logs if this is the last line you see
	source, err := workloadapi.NewX509Source(e.Context)
	if err != nil {
		logrus.Fatalf("error getting x509 source: %+v", err)
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		logrus.Fatalf("error getting x509 svid: %+v", err)
	}
	log.FromContext(e.Context).Infof("SVID: %q", svid.ID)
	e.source = source
}

func (e *Endpoint) register() error {
	// register nse with nsm
	clientOptions := append(
		opentracing.WithTracingDial(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSClientConfig(e.source, e.source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)
	cc, err := grpc.DialContext(e.Context,
		grpcutils.URLToTarget(&e.Config.ConnectTo),
		clientOptions...,
	)
	if err != nil {
		log.FromContext(e.Context).Fatalf("error establishing grpc connection to registry server %+v", err)
	}

	_, err = registryapi.NewNetworkServiceRegistryClient(cc).Register(context.Background(), &registryapi.NetworkService{
		Name:    e.Config.ServiceName,
		Payload: payload.IP,
	})

	if err != nil {
		log.FromContext(e.Context).Fatalf("unable to register ns %+v", err)
	}

	registryClient := registrychain.NewNetworkServiceEndpointRegistryClient(
		registryrefresh.NewNetworkServiceEndpointRegistryClient(),
		registrysendfd.NewNetworkServiceEndpointRegistryClient(),
		registryapi.NewNetworkServiceEndpointRegistryClient(cc),
	)
	nse, err := registryClient.Register(context.Background(), &registryapi.NetworkServiceEndpoint{
		Name:                e.Config.Name,
		NetworkServiceNames: []string{e.Config.ServiceName},
		NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
			e.Config.ServiceName: {
				Labels: e.Config.Labels,
			},
		},
		Url: e.listenOn.String(),
	})
	logrus.Infof("nse: %+v", nse)

	return err
}

func NewEndpoint(context context.Context, config *Config) *Endpoint {
	endpoint := &Endpoint{
		Context: context,
		Config:  config,
	}

	endpoint.setSource()

	return endpoint
}
