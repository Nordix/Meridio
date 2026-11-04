/*
Copyright (c) 2026 OpenInfra Foundation Europe. All rights reserved.

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

// Tests for the add-path monitor timeout (fullmesh addNetworkServiceClient must not
// freeze the recv loop on a hung MonitorConnections stream) and for teardown Close()
// error propagation (SimpleNetworkServiceClient.Close surfaces the leaf close error).
package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// hangingMonitorClient.MonitorConnections blocks until the passed ctx is done, then
// returns ctx.Err(). With a bounded monitorCtx this models a half-open monitor
// socket (e.g. after a node reboot) that would otherwise hang under fmnsc.mu.
type hangingMonitorClient struct {
	entered chan struct{}
}

func (h *hangingMonitorClient) MonitorConnections(
	ctx context.Context, _ *networkservice.MonitorScopeSelector, _ ...grpc.CallOption,
) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	select {
	case <-h.entered:
	default:
		close(h.entered)
	}
	<-ctx.Done() // never delivers an event; unblocks only when monitorCtx is bounded
	return nil, ctx.Err()
}

// TestAddNetworkServiceClient_MonitorTimeoutDoesNotFreezeRecv verifies the add-path
// fix: MonitorConnections + Recv run under fmnsc.mu, so a hung monitor stream must
// be bounded by monitorCtx (config.RequestTimeout) and let addNetworkServiceClient
// return, instead of freezing the recv loop indefinitely.
func TestAddNetworkServiceClient_MonitorTimeoutDoesNotFreezeRecv(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor := &hangingMonitorClient{entered: make(chan struct{})}
	cfg := &Config{
		Name:                    "proxy-test",
		RequestTimeout:          300 * time.Millisecond, // bounds the hung monitor lookup
		MonitorConnectionClient: monitor,
	}
	fmnsc := &FullMeshNetworkServiceClient{
		networkServiceClients:   map[string]NetworkServiceClient{},
		networkServiceEndpoints: map[string]*registry.NetworkServiceEndpoint{},
		teardownInFlight:        map[string]struct{}{},
		client:                  newConnectedNSClient(),
		config:                  cfg,
		ctx:                     ctx,
		logger:                  logr.Discard(),
		baseRequest:             validBaseRequest(),
	}

	// Must return within the monitor timeout plus slack; if it hangs, the bound is
	// missing and the recv loop would freeze.
	returned := completesWithin(2*time.Second, func() {
		fmnsc.addNetworkServiceClient("lb-nse-1")
	})
	require.True(t, returned,
		"addNetworkServiceClient must return once the bounded monitorCtx expires; "+
			"a hung MonitorConnections stream must not freeze the recv loop")

	// The add still proceeds (monitor error is non-fatal): a client is registered.
	fmnsc.mu.Lock()
	_, registered := fmnsc.networkServiceClients["lb-nse-1"]
	fmnsc.mu.Unlock()
	require.True(t, registered, "add should proceed after the monitor lookup times out")
}

// closeErrNSClient connects immediately and fails Close() with a fixed error, so the
// test can assert the error is surfaced (not swallowed) by SimpleNetworkServiceClient.
type closeErrNSClient struct {
	closeErr error
}

func (c *closeErrNSClient) Request(
	_ context.Context, request *networkservice.NetworkServiceRequest, _ ...grpc.CallOption,
) (*networkservice.Connection, error) {
	return request.GetConnection(), nil
}

func (c *closeErrNSClient) Close(
	_ context.Context, _ *networkservice.Connection, _ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return nil, c.closeErr
}

// TestSimpleClientClose_SurfacesLeafError verifies Close() returns (not swallows)
// the leaf close error once a connection was established, so the teardown goroutine
// can log it with context. The retry client retries until RequestTimeout, so the
// test uses a short timeout to keep it fast.
func TestSimpleClientClose_SurfacesLeafError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leafErr := errors.New("boom from leaf Close")
	cfg := &Config{Name: "proxy-test", RequestTimeout: 200 * time.Millisecond}
	leaf := &closeErrNSClient{closeErr: leafErr}

	snsc := NewSimpleNetworkServiceClient(ctx, cfg, leaf)
	req := validBaseRequest()
	req.Connection.NetworkServiceEndpointName = "lb-nse-1"
	require.NoError(t, snsc.Request(req), "request should succeed (connection established)")

	err := snsc.Close()
	require.Error(t, err, "Close() must surface the leaf close error, not swallow it")
	require.Contains(t, err.Error(), "failed to close connection")
}

// TestSimpleClientClose_NilConnectionReturnsNil confirms that when no connection was
// ever established, Close() is a no-op returning nil (nothing to close).
func TestSimpleClientClose_NilConnectionReturnsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &Config{Name: "proxy-test", RequestTimeout: 200 * time.Millisecond}
	snsc := NewSimpleNetworkServiceClient(ctx, cfg, &closeErrNSClient{closeErr: errors.New("unused")})
	require.NoError(t, snsc.Close(), "Close() with no established connection must return nil")
}
