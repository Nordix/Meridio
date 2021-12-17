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

package endpoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/edwarnicke/grpcfd"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/nordix/meridio/pkg/security/credentials"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	grpcCredentials "google.golang.org/grpc/credentials"
)

type Server struct {
	Name                    string
	MaxTokenLifetime        time.Duration
	AdditionalFunctionality []networkservice.NetworkServiceServer
	ctx                     context.Context
	cancel                  context.CancelFunc
	tmpDir                  string
	listenOn                *url.URL
	server                  *grpc.Server
}

func NewServer(name string,
	maxTokenLifetime time.Duration,
	additionalFunctionality []networkservice.NetworkServiceServer) *Server {
	s := &Server{
		Name:                    name,
		MaxTokenLifetime:        maxTokenLifetime,
		AdditionalFunctionality: additionalFunctionality,
	}
	return s
}

func (s *Server) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	source := s.getSource(s.ctx)
	responderEndpoint := endpoint.NewServer(s.ctx,
		spiffejwt.TokenGeneratorFunc(source, s.MaxTokenLifetime),
		endpoint.WithName(s.Name),
		endpoint.WithAuthorizeServer(authorize.NewServer()),
		endpoint.WithAdditionalFunctionality(s.AdditionalFunctionality...))
	options := append(
		opentracing.WithTracing(),
		grpc.Creds(
			grpcfd.TransportCredentials(
				grpcCredentials.NewTLS(
					tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny()),
				),
			),
			// credentials.GetServerWithSource(s.ctx, source),
		),
	)
	s.server = grpc.NewServer(options...)
	responderEndpoint.Register(s.server)
	var err error
	s.tmpDir, err = ioutil.TempDir("", s.Name)
	if err != nil {
		return fmt.Errorf("error creating tmpDir for endpoint server (%v)", err)
	}
	s.listenOn = &(url.URL{Scheme: "unix", Path: filepath.Join(s.tmpDir, "listen.on")})
	srvErrCh := grpcutils.ListenAndServe(s.ctx, s.listenOn, s.server)
	go s.errorHandler(srvErrCh)
	return nil
}

func (s *Server) Stop() error {
	if s.server != nil {
		s.server.Stop()
	}
	if s.cancel != nil {
		s.cancel()
	}
	return os.Remove(s.tmpDir)
}

func (s *Server) GetUrl() string {
	return s.listenOn.String()
}

func (s *Server) errorHandler(errCh <-chan error) {
	err := <-errCh
	logrus.Errorf("err ListenAndServe on NSE server: %v", err)
}

func (s *Server) getSource(ctx context.Context) *workloadapi.X509Source {
	return credentials.GetX509Source(ctx)
}
