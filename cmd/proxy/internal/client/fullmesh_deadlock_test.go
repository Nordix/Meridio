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

// Validation harness for Nordix/Meridio issue #628 ("Proxy stops consuming NSM
// Find(Watch=true) stream").
//
// Unlike a standalone model, this is a WHITE-BOX test in package client: it drives
// the REAL FullMeshNetworkServiceClient.deleteNetworkServiceClient() and the REAL
// SimpleNetworkServiceClient.Request()/Close(), injecting only the leaf
// networkservice.NetworkServiceClient (the thing that, in production, retries
// forever against an NSE that vanished after a node reboot). Because the test
// exercises the actual lock interaction:
//
//	deleteNetworkServiceClient: fmnsc.mu.Lock()                 // (A)
//	  -> SimpleNetworkServiceClient.Close(): requestCancel();   //    signals cancel
//	                                         snsc.mu.Lock()      // (B) blocks on Request
//	  (a concurrent `go Request()` holds snsc.mu while the injected client retries)
//
// it is a genuine regression guard: with the current (faulty) code
// TestDeleteNetworkServiceClient_DoesNotBlockRecvLoop_WhenRequestWedged FAILS
// (deadlock -> timeout); with a fix that stops holding fmnsc.mu across the blocking
// Close() (and/or runs Close asynchronously), it PASSES. See the issue's
// "Suggested fix (direction)".
//
// The injected client's Request models the two regimes the issue distinguishes:
//   - wedged  (ignoreCancel=true):  retry loop never unwinds on ctx cancellation,
//                                   so snsc.mu stays held -> Close() blocks on (B).
//   - healthy (ignoreCancel=false): retry loop returns on cancellation, releasing
//                                   snsc.mu so Close() can proceed (control case).
package client

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// wedgeableNSClient is a networkservice.NetworkServiceClient whose Request blocks
// like the NSM retry client does while the target NSE is unreachable. It is the
// single injected leaf; everything above it in the test is real proxy code.
type wedgeableNSClient struct {
	// requestEntered is closed the first time Request is called, so the test can
	// deterministically wait until the real SimpleNetworkServiceClient.Request has
	// taken snsc.mu (B) before triggering the delete — the exact precondition for
	// the deadlock, established without sleeps.
	requestEntered chan struct{}

	// ignoreCancel selects the regime. true => Request blocks forever regardless of
	// context cancellation (the "does not unwind promptly" case from #628). false =>
	// Request returns when its context is cancelled (healthy retry client).
	ignoreCancel bool

	closed chan struct{}
}

func newWedgeableNSClient(ignoreCancel bool) *wedgeableNSClient {
	return &wedgeableNSClient{
		requestEntered: make(chan struct{}),
		ignoreCancel:   ignoreCancel,
		closed:         make(chan struct{}),
	}
}

func (w *wedgeableNSClient) Request(
	ctx context.Context, _ *networkservice.NetworkServiceRequest, _ ...grpc.CallOption,
) (*networkservice.Connection, error) {
	select {
	case <-w.requestEntered:
	default:
		close(w.requestEntered)
	}
	if w.ignoreCancel {
		// Wedged: block forever, ignoring cancellation — the real SimpleNetwork-
		// ServiceClient.Request keeps holding snsc.mu while this does not return.
		select {}
	}
	// Healthy: unwind as soon as the request context is cancelled, letting the real
	// Request release snsc.mu.
	<-ctx.Done()
	return nil, ctx.Err()
}

func (w *wedgeableNSClient) Close(
	_ context.Context, _ *networkservice.Connection, _ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	select {
	case <-w.closed:
	default:
		close(w.closed)
	}
	return &emptypb.Empty{}, nil
}

// newFullMeshForTest builds a real FullMeshNetworkServiceClient with just the
// fields deleteNetworkServiceClient touches, and pre-registers one real
// SimpleNetworkServiceClient backed by the injected leaf. It then starts that
// client's Request in a goroutine exactly as addNetworkServiceClient does, and
// waits until Request is actually in flight (holding snsc.mu).
func newFullMeshForTest(t *testing.T, ctx context.Context, leaf *wedgeableNSClient) (*FullMeshNetworkServiceClient, string) {
	t.Helper()

	cfg := &Config{
		Name:           "proxy-test",
		RequestTimeout: 2 * time.Second,
	}

	fmnsc := &FullMeshNetworkServiceClient{
		networkServiceClients:   map[string]NetworkServiceClient{},
		networkServiceEndpoints: map[string]*registry.NetworkServiceEndpoint{},
		client:                  leaf,
		config:                  cfg,
		ctx:                     ctx,
		logger:                  logr.Discard(),
		baseRequest:             validBaseRequest(),
	}

	const nseName = "lb-nse-1"

	// Mirror addNetworkServiceClient: construct the REAL SimpleNetworkServiceClient
	// over the injected leaf, register it, and start Request in a goroutine.
	snsc := NewSimpleNetworkServiceClient(ctx, cfg, leaf)
	fmnsc.networkServiceClients[nseName] = snsc

	req := validBaseRequest()
	req.Connection.NetworkServiceEndpointName = nseName
	go func() { _ = snsc.Request(req) }()

	// Precondition: wait until the real Request has entered the injected leaf, i.e.
	// it is holding snsc.mu (B). Without this the delete could win the race and the
	// deadlock would not be reproduced.
	select {
	case <-leaf.requestEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("precondition not met: SimpleNetworkServiceClient.Request never reached the leaf client")
	}

	return fmnsc, nseName
}

func validBaseRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "proxy.load-balancer",
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: "LOCAL", Type: "KERNEL"},
		},
	}
}

// completesWithin reports whether fn returned before the deadline. A true deadlock
// never returns, so this turns the stall into a deterministic assertion.
func completesWithin(d time.Duration, fn func()) bool {
	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

// TestDeleteNetworkServiceClient_DoesNotBlockRecvLoop_WhenRequestWedged is the
// regression guard for issue #628.
//
// EXPECTED AGAINST CURRENT (FAULTY) CODE: FAILS. deleteNetworkServiceClient holds
// fmnsc.mu (A) and calls the real Close() synchronously; Close() cancels the
// request then waits on snsc.mu (B), which the wedged Request never releases, so
// the call never returns — recv() would be deadlocked and fmnsc.mu held.
//
// EXPECTED AGAINST A FIX: PASSES. A fix that does not hold fmnsc.mu across the
// blocking Close() (e.g. remove-under-lock then Close() outside the lock, and/or
// run Close() asynchronously) lets this return promptly and keeps fmnsc.mu free.
func TestDeleteNetworkServiceClient_DoesNotBlockRecvLoop_WhenRequestWedged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leaf := newWedgeableNSClient(true) // wedged: never unwinds on cancel
	fmnsc, nseName := newFullMeshForTest(t, ctx, leaf)

	// The real recv() delete path. It must return promptly; if it blocks, the proxy
	// has stopped draining its Find(Watch=true) stream (the #628 symptom).
	returned := completesWithin(2*time.Second, func() {
		fmnsc.deleteNetworkServiceClient(nseName)
	})
	require.True(t, returned,
		"issue #628: deleteNetworkServiceClient blocked (Close() under fmnsc.mu waiting on "+
			"snsc.mu held by a wedged Request). recv() can no longer drain the Find stream. "+
			"A fix must not hold fmnsc.mu across a blocking Close().")

	// Second consequence from the issue: while the delete path holds fmnsc.mu, every
	// other full-mesh operation stalls too. After the fix returns, the lock must be
	// obtainable. TryLock asserts that directly (no empty critical section) and never
	// blocks the test if the lock is still held by a stalled delete.
	var acquired bool
	lockChecked := completesWithin(1*time.Second, func() {
		for {
			if fmnsc.mu.TryLock() {
				acquired = true
				fmnsc.mu.Unlock()
				return
			}
			time.Sleep(time.Millisecond)
		}
	})
	require.True(t, lockChecked && acquired,
		"fmnsc.mu is still held after deleteNetworkServiceClient returned/blocked, so the whole "+
			"full-mesh client is frozen (add/delete of other endpoints cannot proceed).")
}

// TestDeleteNetworkServiceClient_CompletesWhenRequestUnwinds is the control case.
// With a Request that unwinds on cancellation, even the current synchronous Close()
// completes — proving the deadlock is specific to the "Request wedged" precondition
// from #628, not to the delete path itself. This must pass against BOTH the faulty
// and the fixed code; if it ever fails, the harness (not the product) is wrong.
func TestDeleteNetworkServiceClient_CompletesWhenRequestUnwinds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leaf := newWedgeableNSClient(false) // healthy: unwinds on cancel
	fmnsc, nseName := newFullMeshForTest(t, ctx, leaf)

	returned := completesWithin(5*time.Second, func() {
		fmnsc.deleteNetworkServiceClient(nseName)
	})
	require.True(t, returned,
		"with a Request that unwinds on cancellation, Close() should complete and the delete "+
			"path should return; a stall here indicates a harness error, not the #628 bug.")
}
