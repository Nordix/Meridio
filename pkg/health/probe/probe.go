/*
Copyright (c) 2021-2023 Nordix Foundation

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

package probe

import (
	"context"
	"fmt"
	"time"

	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/log"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

type HealthProbe struct {
	userAgent   string
	addr        string
	service     string
	connTimeout time.Duration
	rpcTimeout  time.Duration
	spiffe      bool
	source      *workloadapi.X509Source
}

// NewHealthProbe -
// Create a new grpc health probe which is an alternate to grpc-health-probe binary
// but does NOT require a new process to get spawned.
// Thus, when using with the spiffe option the spire agents are not required to attest
// new processes all the time when the probe is invoked.
func NewHealthProbe(ctx context.Context, options ...Option) (*HealthProbe, error) {

	opts := &probeOptions{
		rpcTimeout:  time.Second,
		connTimeout: time.Second,
		userAgent:   "meridio-grpc-health-probe",
	}

	for _, opt := range options {
		opt(opts)
	}

	hp := &HealthProbe{
		userAgent:   opts.userAgent,
		addr:        opts.addr,
		service:     opts.service,
		connTimeout: opts.connTimeout,
		rpcTimeout:  opts.rpcTimeout,
		spiffe:      opts.spiffe,
	}

	if hp.spiffe {
		// fetch X509Source once to avoid recurring load on spire
		source, err := workloadapi.NewX509Source(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting x509 source, %w", err)
		}
		svid, err := source.GetX509SVID()
		if err != nil {
			return nil, fmt.Errorf("error getting x509 svid, %w", err)
		}
		log.FromContextOrGlobal(ctx).V(1).Info("GetX509Source for health probe", "sVID", svid.ID)
		hp.source = source
	}

	return hp, nil
}

// String -
func (hp *HealthProbe) String() string {
	return fmt.Sprintf("addr=%v, service=%v, userAgent=%v, connTimeout=%v, rpcTimeout=%v, spiffe=%v",
		hp.addr, hp.service, hp.userAgent, hp.connTimeout.String(), hp.rpcTimeout.String(), hp.spiffe)
}

// Request -
// Sends query to check health of gRPC services exposing their status through
// gRPC Health Checking Protocol.
//
// Returns no error if queried service is SERVING.
func (hp *HealthProbe) Request(ctx context.Context) error {
	opts := []grpc.DialOption{
		grpc.WithUserAgent(hp.userAgent),
		grpc.WithBlock(),
	}

	if hp.spiffe {
		opts = append(opts, grpc.WithTransportCredentials(
			credentials.NewTLS(tlsconfig.MTLSClientConfig(hp.source, hp.source, tlsconfig.AuthorizeAny())),
		))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		))
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, hp.connTimeout)
	defer dialCancel()
	conn, err := grpc.DialContext(dialCtx, hp.addr, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect service %q, %w", hp.addr, err)
	}
	defer conn.Close()

	rpcCtx, rpcCancel := context.WithTimeout(ctx, hp.rpcTimeout)
	defer rpcCancel()
	rpcCtx = metadata.NewOutgoingContext(rpcCtx, make(metadata.MD))
	resp, err := grpc_health_v1.NewHealthClient(conn).Check(
		rpcCtx,
		&grpc_health_v1.HealthCheckRequest{
			Service: hp.service,
		},
	)
	if err != nil {
		return fmt.Errorf("health rpc failed, %w", err)
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service unhealthy, status: %q", resp.GetStatus().String())
	}

	return nil
}

// CreateAndRunGRPCHealthProbe -
// Creates and runs a gRPC health probe client and registers the status response
// of the service to a local gRPC health server retrived from the context.
// In the local health server the service is refered to by healthService string.
func CreateAndRunGRPCHealthProbe(ctx context.Context, healthService string, options ...Option) {
	// create the probe for the service
	ghp, err := NewHealthProbe(ctx, options...)
	if err != nil {
		log.Logger.Error(err, "Failed to create background gRPC Health Probe")
		return
	}
	log.Logger.V(1).Info("Created background probe", "probe", ghp, "health service", healthService)

	runf := func(ctx context.Context) error {
		cancelCtx, cancel := context.WithTimeout(ctx, 4*time.Second) // probe timeout
		defer cancel()
		servingStatus := grpc_health_v1.HealthCheckResponse_SERVING
		if err := ghp.Request(cancelCtx); err != nil {
			servingStatus = grpc_health_v1.HealthCheckResponse_NOT_SERVING
			log.Logger.V(1).Info("Background", "error", err)
		}
		if hs := health.HealthServer(ctx); hs != nil {
			hs.SetServingStatus(healthService, servingStatus)
		}
		return err
	}

	// start probing the service and report serving status to health server if any
	go func() {
		_ = runf(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second): // probe period
			}
			_ = runf(ctx)
		}
	}()
}
