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
//     so snsc.mu stays held -> Close() blocks on (B).
//   - healthy (ignoreCancel=false): retry loop returns on cancellation, releasing
//     snsc.mu so Close() can proceed (control case).
package client

import (
	"context"
	"sync"
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
	requestEntered     chan struct{}
	requestEnteredOnce sync.Once

	// ignoreCancel selects the regime. true => Request blocks forever regardless of
	// context cancellation (the "does not unwind promptly" case). false => Request
	// returns when its context is cancelled (healthy retry client that never
	// connected). See connectImmediately for the connected regime.
	ignoreCancel bool

	// connectImmediately => Request returns a valid connection right away, so the
	// real SimpleNetworkServiceClient stores it and a later Close() reaches this
	// leaf's Close(). Used to observe teardown actually closing the connection.
	connectImmediately bool

	closed     chan struct{}
	closedOnce sync.Once
}

func newWedgeableNSClient(ignoreCancel bool) *wedgeableNSClient {
	return &wedgeableNSClient{
		requestEntered: make(chan struct{}),
		ignoreCancel:   ignoreCancel,
		closed:         make(chan struct{}),
	}
}

func newConnectedNSClient() *wedgeableNSClient {
	return &wedgeableNSClient{
		requestEntered:     make(chan struct{}),
		connectImmediately: true,
		closed:             make(chan struct{}),
	}
}

func (w *wedgeableNSClient) Request(
	ctx context.Context, request *networkservice.NetworkServiceRequest, _ ...grpc.CallOption,
) (*networkservice.Connection, error) {
	w.requestEnteredOnce.Do(func() { close(w.requestEntered) })
	if w.connectImmediately {
		// Healthy + connected: return the request's connection so the real
		// SimpleNetworkServiceClient stores it and Close() later reaches this leaf.
		return request.GetConnection(), nil
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
	w.closedOnce.Do(func() { close(w.closed) })
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
		teardownInFlight:        map[string]struct{}{},
		client:                  leaf,
		config:                  cfg,
		ctx:                     ctx,
		logger:                  logr.Discard(),
		baseRequest:             validBaseRequest(),
	}
	fmnsc.teardownCond = sync.NewCond(&fmnsc.mu)

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

// registerHealthyClient registers a real SimpleNetworkServiceClient for name over
// a fresh healthy leaf (unwinds on cancel), mirroring addNetworkServiceClient, and
// waits until its Request is in flight. Returns the leaf so the test can observe
// Close().
func registerHealthyClient(t *testing.T, ctx context.Context, fmnsc *FullMeshNetworkServiceClient, name string) *wedgeableNSClient {
	t.Helper()
	leaf := newConnectedNSClient()
	snsc := NewSimpleNetworkServiceClient(ctx, fmnsc.config, leaf)

	fmnsc.mu.Lock()
	fmnsc.networkServiceClients[name] = snsc
	fmnsc.mu.Unlock()

	req := validBaseRequest()
	req.Connection.NetworkServiceEndpointName = name
	go func() { _ = snsc.Request(req) }()

	select {
	case <-leaf.requestEntered:
	case <-time.After(3 * time.Second):
		t.Fatalf("precondition not met: Request for %s never reached the leaf", name)
	}
	return leaf
}

// TestClose_DrainsTeardown verifies the shutdown path: Close() snapshots the
// clients, tears each down in its own goroutine, and returns only after every
// teardown finished. With healthy leaves, each leaf's Close() must have run.
func TestClose_DrainsTeardown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmnsc, _ := newFullMeshForTest(t, ctx, newWedgeableNSClient(false))
	// newFullMeshForTest already registered "lb-nse-1" over a healthy leaf; add two more.
	leafA := registerHealthyClient(t, ctx, fmnsc, "lb-nse-A")
	leafB := registerHealthyClient(t, ctx, fmnsc, "lb-nse-B")

	returned := completesWithin(5*time.Second, func() { _ = fmnsc.Close() })
	require.True(t, returned, "Close() must return after all teardown goroutines finish")

	// Both teardowns must have actually invoked the leaf Close().
	for name, leaf := range map[string]*wedgeableNSClient{"lb-nse-A": leafA, "lb-nse-B": leafB} {
		select {
		case <-leaf.closed:
		case <-time.After(time.Second):
			t.Fatalf("Close() returned but teardown never closed leaf %s", name)
		}
	}

	// No clients and no in-flight teardown remain.
	fmnsc.mu.Lock()
	nClients := len(fmnsc.networkServiceClients)
	nTeardown := len(fmnsc.teardownInFlight)
	fmnsc.mu.Unlock()
	require.Equal(t, 0, nClients, "all clients should be removed after Close()")
	require.Equal(t, 0, nTeardown, "no teardown should remain in flight after Close()")
}

// TestClose_ConcurrentDeleteDoesNotPanicWaitGroup is the regression guard for the
// "WaitGroup is reused before previous Wait has returned" panic. The recv loop
// stays live during Close() (shutdown cancels the context after Close() in
// production), so deletes can start new teardown while Close() is waiting. The
// teardown accounting must tolerate that without panicking.
func TestClose_ConcurrentDeleteDoesNotPanicWaitGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmnsc, _ := newFullMeshForTest(t, ctx, newWedgeableNSClient(false))

	// Pre-register many healthy clients.
	names := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		name := "lb-nse-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		registerHealthyClient(t, ctx, fmnsc, name)
		names = append(names, name)
	}

	// Concurrently: Close() (drains teardown) and a recv-like deleter hammering
	// deleteNetworkServiceClient. Pre-fix this panics in Close()'s WaitGroup.Wait.
	done := completesWithin(10*time.Second, func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, name := range names {
				fmnsc.deleteNetworkServiceClient(name)
			}
		}()
		_ = fmnsc.Close()
		wg.Wait()
	})
	require.True(t, done, "Close() racing concurrent deletes must not block or panic")

	fmnsc.mu.Lock()
	nTeardown := len(fmnsc.teardownInFlight)
	fmnsc.mu.Unlock()
	require.Equal(t, 0, nTeardown, "no teardown should remain in flight")
}

// TestReAddSameID_DoesNotRunConcurrentlyWithTeardown is the regression guard for
// the detached-teardown-vs-readd race. While a wedged Close() for an endpoint is
// in flight, a re-add of the same name (same deterministic NSM connection id) must
// NOT register a new client — otherwise its Request() would act on the same
// connection concurrently with the old Close().
func TestReAddSameID_DoesNotRunConcurrentlyWithTeardown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wedged leaf: its teardown Close() will not finish, so the name stays in
	// teardownInFlight for the duration of the test.
	leaf := newWedgeableNSClient(true)
	fmnsc, nseName := newFullMeshForTest(t, ctx, leaf)
	// TeardownWaitTimeout == 0 is the documented disable (skip-immediately) path.
	// Make it explicit rather than relying on the zero value.
	fmnsc.config.TeardownWaitTimeout = 0

	// Start teardown; it returns promptly (detached) but leaves teardown in flight.
	returned := completesWithin(2*time.Second, func() {
		fmnsc.deleteNetworkServiceClient(nseName)
	})
	require.True(t, returned, "delete must return promptly even with a wedged Close()")

	// The name must be marked in-flight.
	fmnsc.mu.Lock()
	_, inFlight := fmnsc.teardownInFlight[nseName]
	fmnsc.mu.Unlock()
	require.True(t, inFlight, "teardown of the endpoint should be in flight")

	// With the wait disabled (timeout 0), a re-add of the same name must skip
	// *immediately* (no new client) while teardown runs, and must not block. A tight
	// bound asserts it does not fall through to any wait path.
	addReturned := completesWithin(50*time.Millisecond, func() {
		fmnsc.addNetworkServiceClient(nseName)
	})
	require.True(t, addReturned, "with timeout 0, re-add must skip immediately, not block")

	fmnsc.mu.Lock()
	_, registered := fmnsc.networkServiceClients[nseName]
	fmnsc.mu.Unlock()
	require.False(t, registered,
		"re-add must not register a new client while the old one's Close() is still in flight "+
			"(same deterministic connection id)")
}

// TestDeleteNetworkServiceClient_DoubleDeleteAndUnknown verifies the delete path is
// idempotent: deleting an unknown name is a no-op, and deleting the same name twice
// spawns exactly one teardown (the second call finds it already removed).
func TestDeleteNetworkServiceClient_DoubleDeleteAndUnknown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmnsc, _ := newFullMeshForTest(t, ctx, newWedgeableNSClient(false))
	leaf := registerHealthyClient(t, ctx, fmnsc, "lb-nse-X")

	// Unknown name: no panic, no teardown.
	fmnsc.deleteNetworkServiceClient("does-not-exist")
	fmnsc.mu.Lock()
	_, unknownInFlight := fmnsc.teardownInFlight["does-not-exist"]
	fmnsc.mu.Unlock()
	require.False(t, unknownInFlight, "deleting an unknown endpoint must not create teardown")

	// First delete tears down; second finds nothing.
	fmnsc.deleteNetworkServiceClient("lb-nse-X")
	fmnsc.deleteNetworkServiceClient("lb-nse-X")

	// Exactly one teardown ran: the leaf Close() fires once and drains.
	select {
	case <-leaf.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("first delete should have torn the connection down")
	}
	require.True(t, completesWithin(2*time.Second, func() { _ = fmnsc.Close() }),
		"Close() must drain the single in-flight teardown")
}

// gatedCloseNSClient connects immediately but blocks its Close() until gate is
// closed, so a test can control exactly when a teardown finishes. Used to prove
// the block-with-timeout re-add path: a re-add that enters waitForTeardown wakes
// and registers once teardown completes within the timeout.
type gatedCloseNSClient struct {
	requestEntered     chan struct{}
	requestEnteredOnce sync.Once
	gate               chan struct{}
}

func newGatedCloseNSClient(gate chan struct{}) *gatedCloseNSClient {
	return &gatedCloseNSClient{requestEntered: make(chan struct{}), gate: gate}
}

func (g *gatedCloseNSClient) Request(
	_ context.Context, request *networkservice.NetworkServiceRequest, _ ...grpc.CallOption,
) (*networkservice.Connection, error) {
	g.requestEnteredOnce.Do(func() { close(g.requestEntered) })
	return request.GetConnection(), nil
}

func (g *gatedCloseNSClient) Close(
	_ context.Context, _ *networkservice.Connection, _ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	<-g.gate
	return &emptypb.Empty{}, nil
}

// registerGatedClient mirrors addNetworkServiceClient's registration, backed by a
// gated leaf so the test controls teardown completion.
func registerGatedClient(t *testing.T, ctx context.Context, fmnsc *FullMeshNetworkServiceClient, name string, gate chan struct{}) {
	t.Helper()
	leaf := newGatedCloseNSClient(gate)
	snsc := NewSimpleNetworkServiceClient(ctx, fmnsc.config, leaf)
	fmnsc.mu.Lock()
	fmnsc.networkServiceClients[name] = snsc
	fmnsc.mu.Unlock()
	req := validBaseRequest()
	req.Connection.NetworkServiceEndpointName = name
	go func() { _ = snsc.Request(req) }()
	select {
	case <-leaf.requestEntered:
	case <-time.After(3 * time.Second):
		t.Fatalf("precondition not met: Request for %s never reached the leaf", name)
	}
}

// TestReAddSameID_WaitsThenRegistersWhenTeardownFinishes is the guard for the
// block-with-timeout behavior: when a teardown of the same endpoint finishes
// within TeardownWaitTimeout, a concurrent re-add must WAIT for it and then
// register a fresh client (not skip). This is the behavioral change vs the
// skip-only variant.
func TestReAddSameID_WaitsThenRegistersWhenTeardownFinishes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmnsc, _ := newFullMeshForTest(t, ctx, newWedgeableNSClient(false))
	fmnsc.config.TeardownWaitTimeout = 3 * time.Second

	const name = "lb-nse-gated"
	gate := make(chan struct{})
	registerGatedClient(t, ctx, fmnsc, name, gate)

	// Start teardown; its Close() blocks on the gate, so the name stays in
	// teardownInFlight until we release it.
	fmnsc.deleteNetworkServiceClient(name)
	fmnsc.mu.Lock()
	_, inFlight := fmnsc.teardownInFlight[name]
	fmnsc.mu.Unlock()
	require.True(t, inFlight, "teardown should be in flight (gated Close not released)")

	// Concurrently re-add the same name: it must enter waitForTeardown and block.
	addDone := make(chan struct{})
	go func() {
		fmnsc.addNetworkServiceClient(name)
		close(addDone)
	}()

	// The re-add must NOT complete while teardown is still in flight.
	select {
	case <-addDone:
		t.Fatal("re-add returned before teardown finished; it should have waited")
	case <-time.After(300 * time.Millisecond):
		// still waiting — expected
	}

	// Release teardown: Close() returns, the detached goroutine clears the mark and
	// broadcasts, waking the waiting re-add.
	close(gate)

	select {
	case <-addDone:
	case <-time.After(3 * time.Second):
		t.Fatal("re-add did not complete after teardown finished within the timeout")
	}

	fmnsc.mu.Lock()
	_, registered := fmnsc.networkServiceClients[name]
	_, stillInFlight := fmnsc.teardownInFlight[name]
	fmnsc.mu.Unlock()
	require.True(t, registered, "re-add must register a fresh client after teardown finished within timeout")
	require.False(t, stillInFlight, "teardown mark must be cleared after Close() finished")
}

// TestReAddSameID_TimesOutAndSkipsWhenTeardownWedged is the guard for the bounded
// fallback: if teardown is stuck past TeardownWaitTimeout, the re-add must time
// out and skip (no new client registered), keeping the recv loop from stalling
// indefinitely (issue #628 guarantee).
func TestReAddSameID_TimesOutAndSkipsWhenTeardownWedged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	leaf := newWedgeableNSClient(true) // wedged: teardown Close() never finishes
	fmnsc, nseName := newFullMeshForTest(t, ctx, leaf)
	fmnsc.config.TeardownWaitTimeout = 200 * time.Millisecond

	fmnsc.deleteNetworkServiceClient(nseName)

	// Re-add must return within a bounded time (timeout + slack), having skipped.
	addReturned := completesWithin(2*time.Second, func() {
		fmnsc.addNetworkServiceClient(nseName)
	})
	require.True(t, addReturned, "re-add must return after the wait times out (recv loop must not stall)")

	fmnsc.mu.Lock()
	_, registered := fmnsc.networkServiceClients[nseName]
	fmnsc.mu.Unlock()
	require.False(t, registered,
		"re-add must skip (no new client) when teardown is still in flight after the timeout")
}
