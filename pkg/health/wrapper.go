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

package health

import (
	"context"
	"errors"

	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// CreateChecker -
// Wraps NewChecker() and saves Checker into context, also starts Checker in a goroutine
// TODO: revisit if it's really needed to fail the program on error
func CreateChecker(ctx context.Context, options ...Option) context.Context {
	opts := append([]Option{
		WithCtx(ctx),
	}, options...)

	healthChecker, err := NewChecker(opts...)
	if err != nil {
		log.Fatal(log.Logger, "Unable to create Health checker", "error", err)
		return ctx
	}

	go func() {
		err := healthChecker.Start()
		if err != nil {
			log.Fatal(log.Logger, "Unable to start Health checker", "error", err)
			return
		}
	}()

	return WithHealthServer(ctx, healthChecker)
}

// RegisterReadinessSubservices -
// Wraps Checker.RegisterServices() by fetching the health server (i.e. Checker) from context
func RegisterReadinessSubservices(ctx context.Context, services ...string) error {
	if h := HealthServer(ctx); h != nil {
		h.RegisterServices(Readiness, services...)
		return nil
	}
	return errors.New("no health server in context")
}

// RegisterLivenessSubservices -
// Wraps Checker.RegisterServices() by fetching the health server (i.e. Checker) from context
func RegisterLivenessSubservices(ctx context.Context, services ...string) error {
	if h := HealthServer(ctx); h != nil {
		h.RegisterServices(Liveness, services...)
		return nil
	}
	return errors.New("no health server in context")
}

// SetServingStatus -
// Wraps Checker.SetServingStatus() by fetching the health server (i.e. Checker) from context
func SetServingStatus(ctx context.Context, service string, serving bool) {
	if h := HealthServer(ctx); h != nil {
		status := grpc_health_v1.HealthCheckResponse_NOT_SERVING
		if serving {
			status = grpc_health_v1.HealthCheckResponse_SERVING
		}
		h.SetServingStatus(service, status)
	}
}
