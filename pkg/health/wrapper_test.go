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

package health_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/health/probe"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var readinessServices []string = []string{"service-1", "service-2", "service-3", "service-4"}
var livenessServices []string = []string{"service-1", "service-5"}

func healthClient(ctx context.Context, address url.URL) error {
	hp, err := probe.NewHealthProbe(ctx, probe.WithAddress(address.String()))
	if err != nil {
		return fmt.Errorf("failed to create health client, %v", err)
	}

	probeCtx, probeCancel := context.WithTimeout(ctx, 1*time.Second)
	defer probeCancel()
	if err := hp.Request(probeCtx); err != nil {
		return fmt.Errorf("health client request failed: %v", err)
	}

	return nil
}

func TestCreateChecker(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	// the returned context must contain health server i.e. the checker created above
	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestSetServingStatus(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	var healthService string = "random-service"
	var resp *grpc_health_v1.HealthCheckResponse
	// health service not registered to the health server shall return error upon Check()
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: healthService})
	require.Error(t, err)
	require.Nil(t, resp)

	// set health service status to NOT_SERVING while also registering it at the health server
	health.SetServingStatus(ctx, healthService, false)
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: healthService})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// change health service status to SERVING
	health.SetServingStatus(ctx, healthService, true)
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: healthService})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestRegisterReadinessSubservices_NewSubservices(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// register subservices for Readiness probe
	err = health.RegisterReadinessSubservices(ctx, readinessServices...)
	require.NoError(t, err)

	var resp *grpc_health_v1.HealthCheckResponse
	// initial probe status must be NOT_SERVING because its subservices
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// change status of subservices to SERVING
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, true)
	}
	// probe service status must be SERVING
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// set status of 1 subservice to NOT_SERVING
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, false)
		break
	}
	// probe service must be NOT_SERVING due to the above status change
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestRegisterReadinessSubservices_ExistingServingSubservices(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// register subservice with SERVING status
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, true)
	}

	// register above subservices for Readiness probe
	err = health.RegisterReadinessSubservices(ctx, readinessServices...)
	require.NoError(t, err)

	var resp *grpc_health_v1.HealthCheckResponse
	// initial probe status must be SERVING because its subservices
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// set status of 1 subservice to NOT_SERVING
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, false)
		break
	}
	// probe service must be NOT_SERVING due to the above status change
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestRegisterReadinessSubservices_ExistingNotServingSubservices(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// register subservice with NOT_SERVING status
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, true)
	}

	// register above subservices for Readiness probe
	err = health.RegisterReadinessSubservices(ctx, readinessServices...)
	require.NoError(t, err)

	var resp *grpc_health_v1.HealthCheckResponse
	// initial probe status must be NOT_SERVING because its subservices
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// change status of subservices to SERVING
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, true)
	}
	// probe service status must be SERVING
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// set status of 1 subservice to NOT_SERVING
	for _, service := range readinessServices {
		health.SetServingStatus(ctx, service, false)
		break
	}
	// probe service must be NOT_SERVING due to the above status change
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestRegisterLivenessSubservices_NewSubservices(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// register subservices for Liveness probe
	err = health.RegisterLivenessSubservices(ctx, livenessServices...)
	require.NoError(t, err)

	var resp *grpc_health_v1.HealthCheckResponse
	// initial probe status must be NOT_SERVING because its subservices
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Liveness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// change status of subservices to SERVING
	for _, service := range livenessServices {
		health.SetServingStatus(ctx, service, true)
	}
	// probe service status must be SERVING
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Liveness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// set status of 1 subservice to NOT_SERVING
	for _, service := range livenessServices {
		health.SetServingStatus(ctx, service, false)
		break
	}
	// probe service must be NOT_SERVING due to the above status change
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Liveness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestRegisterLivenessSubservices_EmptySubservices(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	emptyLivenessServices := []string{}
	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// register subservices for Liveness probe
	err = health.RegisterLivenessSubservices(ctx, emptyLivenessServices...)
	require.NoError(t, err)

	var resp *grpc_health_v1.HealthCheckResponse
	// initial probe status must be SERVING because there are no subservices
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Liveness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// can change probe status directly because there are no subservices
	health.SetServingStatus(ctx, health.Liveness, false)
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Liveness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}

func TestRegisterLivenessSubservices_ExistingSharedSubservices(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	var sharedServices []string = []string{"service-1", "service-2", "service-3"}
	readinessServices := sharedServices
	livenessServices := sharedServices
	var resp *grpc_health_v1.HealthCheckResponse
	dir, err := os.MkdirTemp(os.TempDir(), t.Name())
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	socket := path.Join(dir, "folder", "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	serverAddr := &url.URL{Scheme: "unix", Path: socket}
	ctx = health.CreateChecker(ctx, health.WithURL(serverAddr))

	hs := health.HealthServer(ctx)
	require.NotNil(t, hs)

	// register shared subservices with SERVING status
	for _, service := range sharedServices {
		health.SetServingStatus(ctx, service, true)
	}

	// register readiness subservices for Liveness probe
	err = health.RegisterReadinessSubservices(ctx, readinessServices...)
	require.NoError(t, err)
	_, _ = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})

	// register liveness subservices for Liveness probe
	err = health.RegisterLivenessSubservices(ctx, livenessServices...)
	require.NoError(t, err)

	// initial liveness probe status must be SERVING because the subservices are SERVING
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Liveness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// readiness probe status must be SERVING
	resp, err = hs.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: health.Readiness})
	require.NoError(t, err)
	require.Equal(t, resp.Status, grpc_health_v1.HealthCheckResponse_SERVING)

	// use health client to secure health server is serving by the time cancel is called
	err = healthClient(ctx, *serverAddr)
	require.NoError(t, err)

	cancel()
	<-ctx.Done()
}
