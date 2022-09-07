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

package probe

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type GrpcHealthProbe struct {
	cmd         string
	addr        string
	service     string
	connTimeout string
	rpcTimeout  string
	spiffe      string
}

// NewGRPCHealthProbe -
// Creates gRPC health checking probe
func NewGRPCHealthProbe(options ...Option) (*GrpcHealthProbe, error) {
	opts := &probeOptions{
		cmd:         "grpc_health_probe",
		addr:        fmt.Sprintf("-addr=%v", health.DefaultURL),
		service:     "-service=",
		rpcTimeout:  "-rpc-timeout=350ms",
		connTimeout: "-connect-timeout=250ms",
	}
	for _, opt := range options {
		opt(opts)
	}
	_, err := exec.LookPath(opts.cmd)
	if err != nil {
		return nil, errors.New("GrpcHealthProbe not found, err: " + err.Error())
	}

	return &GrpcHealthProbe{
		cmd:         opts.cmd,
		addr:        opts.addr,
		service:     opts.service,
		rpcTimeout:  opts.rpcTimeout,
		connTimeout: opts.connTimeout,
		spiffe:      opts.spiffe,
	}, nil
}

// String -
func (ghp *GrpcHealthProbe) String() string {
	return fmt.Sprintf("%v %v %v %v %v %v", ghp.cmd, ghp.addr, ghp.service, ghp.rpcTimeout, ghp.rpcTimeout, ghp.spiffe)
}

// Run -
// Runs gRPC health checking probe
// Returns nil if status is HealthCheckResponse_SERVING, and err otherwise
func (ghp *GrpcHealthProbe) Run(ctx context.Context) error {
	stdoutStderr, err := exec.CommandContext(ctx, ghp.cmd, ghp.addr, ghp.service,
		ghp.connTimeout, ghp.rpcTimeout, ghp.spiffe).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gRPC Health Probe err: %v, %s", err, stdoutStderr)
	}

	return nil
}

// CreateAndRunGRPCHealthProbe -
// Creates gRPC Health Probe and starts running it background while registering
// the probing results to Health Server.
// TODO: configurable period, timeout
func CreateAndRunGRPCHealthProbe(ctx context.Context, healthService string, options ...Option) {
	// create the probe for the servie
	ghp, err := NewGRPCHealthProbe(options...)
	if err != nil {
		log.Logger.Error(err, "Failed to create background gRPC Health Probe")
		return
	}
	log.Logger.V(1).Info("Created background probe", "probe", ghp)

	runf := func(ctx context.Context) error {
		cancelCtx, cancel := context.WithTimeout(ctx, 3*time.Second) // probe timeout
		defer cancel()
		servingStatus := grpc_health_v1.HealthCheckResponse_SERVING
		if err := ghp.Run(cancelCtx); err != nil {
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
