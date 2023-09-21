/*
Copyright (c) 2023 Nordix Foundation

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

package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nordix/meridio/pkg/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

type Server struct {
	IP   string
	Port int
}

func (s *Server) Start(ctx context.Context) error {
	log.FromContextOrGlobal(ctx).Info("Start metrics server", "ip", s.IP, "port", s.Port)

	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions())
	if err != nil {
		return err
	}
	defer source.Close()

	server := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", s.IP, s.Port),
		TLSConfig: tlsconfig.TLSServerConfig(source),
	}

	http.Handle("/metrics", promhttp.Handler())

	err = server.ListenAndServeTLS("", "")
	if err != nil {
		return err
	}

	return nil
}
