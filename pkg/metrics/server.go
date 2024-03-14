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
	"time"

	"github.com/nordix/meridio/pkg/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	IP   string
	Port int
}

func (s *Server) Start(ctx context.Context) error {
	log.FromContextOrGlobal(ctx).Info("Start metrics server", "ip", s.IP, "port", s.Port)

	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", s.IP, s.Port),
	}

	http.Handle("/metrics", promhttp.Handler())

	serverCtx, cancel := context.WithCancel(ctx)
	var ListenAndServeErr error

	go func() {
		ListenAndServeErr = server.ListenAndServe()
		if ListenAndServeErr != nil {
			cancel()
		}
	}()

	<-serverCtx.Done()

	if ListenAndServeErr != nil {
		return fmt.Errorf("failed to ListenAndServe on metrics server: %w", ListenAndServeErr)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	err := server.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("failed to shutdown metrics server: %w", err)
	}

	return nil
}
