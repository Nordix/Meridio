/*
Copyright (c) 2025 OpenInfra Foundation Europe

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

package stream_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	lbAPI "github.com/nordix/meridio/api/loadbalancer/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/stream"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testPathName     = "test-path"
	testPathHostname = "test-path-hostname"
)

var forwardingAvailabilityTarget = &lbAPI.Target{Context: map[string]string{types.IdentifierKey: testPathHostname}}

// --- Test Helpers and Mocks ---

// mockStreamAvailabilityService_WatchServer is a test double for the gRPC WatchServer interface
type mockStreamAvailabilityService_WatchServer struct {
	lbAPI.StreamAvailabilityService_WatchServer
	ctx       context.Context
	responses chan *lbAPI.Response // Channel to send responses back to the test goroutine
	sendDelay time.Duration        // Simulate network latency for Send() calls
}

func (m *mockStreamAvailabilityService_WatchServer) Context() context.Context {
	return m.ctx
}

// Send implements the WatchServer interface, simulating sending a response.
// It includes a delay to simulate network processing.
func (m *mockStreamAvailabilityService_WatchServer) Send(resp *lbAPI.Response) error {
	// First, simulate network delay
	select {
	case <-time.After(m.sendDelay):
	case <-m.ctx.Done():
		return m.ctx.Err()
	}

	// Send the response to the test's channel
	select {
	case m.responses <- resp:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

// Implement other methods required by the interface
func (m *mockStreamAvailabilityService_WatchServer) RecvMsg(v interface{}) error {
	return fmt.Errorf("mock RecvMsg not implemented")
}
func (m *mockStreamAvailabilityService_WatchServer) SendMsg(v interface{}) error {
	return fmt.Errorf("mock SendMsg not implemented")
}
func (m *mockStreamAvailabilityService_WatchServer) Header() (md metadata.MD, err error) {
	return metadata.MD{}, nil
}
func (m *mockStreamAvailabilityService_WatchServer) Trailer() metadata.MD           { return metadata.MD{} }
func (m *mockStreamAvailabilityService_WatchServer) SetHeader(md metadata.MD) error { return nil }
func (m *mockStreamAvailabilityService_WatchServer) SetTrailer(md metadata.MD)      {}

// newMockWatcherServer creates a configured mock WatchServer for testing
func newMockWatcherServer(ctx context.Context, sendDelay time.Duration) (*mockStreamAvailabilityService_WatchServer, chan *lbAPI.Response) {
	responses := make(chan *lbAPI.Response, 10) // Buffered channel for responses
	mockServer := &mockStreamAvailabilityService_WatchServer{
		ctx:       ctx,
		responses: responses,
		sendDelay: sendDelay,
	}
	return mockServer, responses
}

// watcherClientInfo holds all relevant data for a single mock watcher client
type watcherClientInfo struct {
	id         int
	ctx        context.Context
	cancel     context.CancelFunc
	mockServer *mockStreamAvailabilityService_WatchServer
	responses  chan *lbAPI.Response
	doneCh     chan struct{} // Signals when the watcher goroutine has exited
}

// newForwardingAvailabilityServiceForTest initializes a ForwardingAvailabilityService
// instance using the test-specific constructor
func newForwardingAvailabilityServiceForTest(
	t *testing.T,
	ctx context.Context,
	forwardingAvailabilityTarget *lbAPI.Target,
	delay time.Duration,
) *stream.ForwardingAvailabilityService {
	t.Helper() // Mark this as a test helper

	fas := stream.NewForwardingAvailabilityServiceForTest(ctx, forwardingAvailabilityTarget, delay)
	require.NotNil(t, fas, "ForwardingAvailabilityService should be created")
	return fas
}

// assertInitialWatcherResponse waits for and verifies the initial empty target response
func assertInitialWatcherResponse(t *testing.T, ctx context.Context, responses <-chan *lbAPI.Response, watcherID string) {
	t.Helper() // Marks this as a test helper

	select {
	case resp, ok := <-responses:
		assert.True(t, ok, "Watcher %s should receive initial target response", watcherID)
		require.NotNil(t, resp, "Initial watcher %s response should not be nil", watcherID)
		assert.Len(t, resp.GetTargets(), 1, "Watcher %s should receive empty initial target", watcherID)
		assert.True(t, len(resp.GetTargets()[0].GetContext()) == 0, "Initial watcher %s target should be empty (zero context fields)", watcherID)
		t.Logf("Initial watcher %s response (empty target) received successfully", watcherID)
	case <-ctx.Done():
		t.Fatalf("Test context cancelled before initial watcher %s response received: %v", watcherID, ctx.Err())
	case <-time.After(1 * time.Second): // A generous timeout for initial setup
		t.Fatalf("Timeout waiting for initial watcher %s response during setup", watcherID)
	}
}

// assertTargetUpdate waits for and verifies an update with the test target
func assertTargetUpdate(t *testing.T, ctx context.Context, responses <-chan *lbAPI.Response, watcherID, expectedHostname string) {
	t.Helper()

	select {
	case resp, ok := <-responses:
		assert.True(t, ok, "Watcher %s should receive updated target response", watcherID)
		require.NotNil(t, resp, "Watcher %s response should not be nil", watcherID)
		require.Len(t, resp.GetTargets(), 1, "Watcher %s should receive one target", watcherID)
		id, ok := (resp.GetTargets()[0].GetContext())[types.IdentifierKey]
		assert.True(t, ok, "Watcher %s target context should contain IdentifierKey", watcherID)
		assert.Equal(t, expectedHostname, id, "Watcher %s target hostname mismatch", watcherID)
		t.Logf("Watcher %s received updated target (%s) successfully", watcherID, expectedHostname)
	case <-ctx.Done():
		t.Fatalf("Test context cancelled before watcher %s updated response received: %v", watcherID, ctx.Err())
	case <-time.After(1 * time.Second): // A generous timeout for updates to propagate
		t.Fatalf("Timeout waiting for watcher %s updated response", watcherID)
	}
}

// --- Test Cases ---

func Test_DeadlockScenario_WithDelayedGetTarget(t *testing.T) {
	testGetTargetDelay := 50 * time.Millisecond // Simulate delay in getTarget() for contention
	sendDelay := 5 * time.Millisecond           // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second              // Overall test timeout

	testCtx, cancelTest := context.WithTimeout(context.Background(), testTimeout)
	defer cancelTest()

	_ = log.New("test", "INFO") // Initialize the logger (optional, but can help following the test run depending on the log level)

	// Initialize the ForwardingAvailabilityService with the test-specific constructor
	fas := newForwardingAvailabilityServiceForTest(t, testCtx, forwardingAvailabilityTarget, testGetTargetDelay)

	// --- Step 1: Start watcher client and verify initial state ---
	t.Log("Starting watcher client...")
	watcherClientCtx, watcherCancel := context.WithCancel(testCtx) // Context for this specific watcher
	defer watcherCancel()

	mockWatcherServer, watcherResponses := newMockWatcherServer(watcherClientCtx, sendDelay)
	require.NotNil(t, mockWatcherServer, "mockWatcherServer should be created")

	// Start the watcher goroutine
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		err := fas.Watch(&emptypb.Empty{}, mockWatcherServer) // Inside this Watch call, fas.getTarget() should now be delayed
		if err != nil && err != watcherClientCtx.Err() && err != testCtx.Err() {
			t.Errorf("Watcher exited with unexpected error: %v", err)
		}
	}()

	// Verify its initial state
	assertInitialWatcherResponse(t, testCtx, watcherResponses, "1")

	// --- Step 2: Trigger concurrent Register/Unregister operations ---
	// Unregister routine waits for Register to conclude to secure wanted
	// execution order. Aim is to provoke contention for `fas.mu.Lock()`
	// while the watcher is potentially delayed in its `getTarget()` call.
	t.Logf("Triggering concurrent Register/Unregister operations...")
	var wgRegister sync.WaitGroup
	var wgUnregister sync.WaitGroup

	wgRegister.Add(1)
	go func() {
		defer wgRegister.Done()
		fas.Register(testPathName)
	}()

	wgUnregister.Add(1)
	go func(wgRegister *sync.WaitGroup) {
		defer wgUnregister.Done()
		wgRegister.Wait() // Ensure Register completes first
		fas.Unregister(testPathName)
	}(&wgRegister)

	// --- Step 3: Verify responses after concurrent operations  ---
	// Note: The test will hang if a deadlock occurred. The testCtx timeout
	// should catch it.
	//
	// Expect some response from watcher upon Register/Unregister.
	// The response might be either the empty Target or the test Target depending
	// on if both calls or only the Register has concluded.
	t.Log("Waiting for watcher responses after Register/Unregister...")

	// First response (either registered or already unregistered)
	select {
	case resp, ok := <-watcherResponses:
		assert.True(t, ok, "Watcher should receive response after Register/Unregister")
		require.NotNil(t, resp, "Response after Register/Unregister should not be nil")
		assert.Len(t, resp.GetTargets(), 1, "Response should contain one target")

		if resp.GetTargets()[0].GetContext() == nil || len(resp.GetTargets()[0].GetContext()) == 0 {
			t.Logf("Watcher response for Register-Unregister (empty target) received")
			// This means both Register and Unregister concluded quickly. No further response expected.
		} else {
			// Only Register processed, expect the full target.
			id, ok := (resp.GetTargets()[0].GetContext())[types.IdentifierKey]
			assert.True(t, ok, "Target context should contain IdentifierKey")
			assert.Equal(t, id, testPathHostname, "Target hostname mismatch after Register")
			t.Logf("Watcher response for Register (test target) received")
		}
	case <-testCtx.Done():
		t.Fatalf("Test context cancelled before first watcher response for Register/Unregister: %v", testCtx.Err())
	case <-time.After(1 * time.Second): // Generous timeout
		t.Fatal("Timeout waiting for first watcher response after Register/Unregister")
	}

	// Note: Currently Watch is not smart enough to avoid sending out the same availability information
	// twice. So irrespective of the availability target received above, wait for another one only to
	// avoid a potential Send() error in the Watcher.
	t.Log("Waiting for watcher response after Unregister...")
	select {
	case resp, ok := <-watcherResponses:
		assert.True(t, ok, "Watcher should receive response after Unregister")
		require.NotNil(t, resp, "Response after Unregister should not be nil")
		assert.Len(t, resp.GetTargets(), 1, "Response after Unregister should contain one target")
		assert.True(t, len(resp.GetTargets()[0].GetContext()) == 0, "Target after Unregister should be empty")
		t.Logf("Watcher response for Unregister (empty target) received")
	case <-testCtx.Done():
		t.Fatalf("Test context cancelled before watcher response for Unregister received: %v", testCtx.Err())
	case <-time.After(1 * time.Second): // Generous timeout
		t.Fatal("Timeout waiting for watcher response after Unregister")
	}

	// --- Step 4: Final synchronization and shutdown  ---
	t.Log("Waiting for all Register/Unregister goroutines to complete...")
	wgUnregister.Wait() // Waits for Unregister, which in turn waits for Register
	t.Log("All Register/Unregister goroutines completed")

	t.Log("Cancelling main test context to trigger service shutdown...")
	cancelTest() // Triggers ctx.Done() for watcher and fas internally

	t.Log("Waiting for watcher goroutine to exit gracefully...")
	select {
	case <-watcherDone:
		t.Log("Watcher goroutine exited gracefully after test cancellation")
	case <-time.After(1 * time.Second):
		t.Error("Watcher goroutine did not exit gracefully after test cancellation. Potential resource leak.")
	}
}

func Test_TwoWatchers_Register(t *testing.T) {
	const numWatchers = 2 // watchers to test with

	testGetTargetDelay := 0 * time.Millisecond // No delay in getTarget
	sendDelay := 5 * time.Millisecond          // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second             // Overall test timeout

	testCtx, cancelTest := context.WithTimeout(context.Background(), testTimeout)
	defer cancelTest()

	_ = log.New("test", "INFO") // Initialize logger

	// Initialize the ForwardingAvailabilityService with the test-specific constructor
	fas := newForwardingAvailabilityServiceForTest(t, testCtx, forwardingAvailabilityTarget, testGetTargetDelay)

	// --- 1. Start watcher clients in a loop and verify initial states ---
	var watchers []*watcherClientInfo // Slice to store info for all watchers

	for i := 1; i <= numWatchers; i++ {
		t.Logf("Setting up Watcher %d client...", i)
		watcherClientCtx, watcherCancel := context.WithCancel(testCtx)
		mockServer, responses := newMockWatcherServer(watcherClientCtx, sendDelay)
		doneCh := make(chan struct{})

		info := &watcherClientInfo{
			id:         i,
			ctx:        watcherClientCtx,
			cancel:     watcherCancel,
			mockServer: mockServer,
			responses:  responses,
			doneCh:     doneCh,
		}
		watchers = append(watchers, info)

		// Start the watcher goroutine
		go func(info *watcherClientInfo) {
			defer close(info.doneCh)
			err := fas.Watch(&emptypb.Empty{}, info.mockServer)
			if err != nil && err != info.ctx.Err() && err != testCtx.Err() {
				t.Errorf("Watcher %d exited with unexpected error: %v", info.id, err)
			}
		}(info)

		// Verify its initial state
		assertInitialWatcherResponse(t, testCtx, info.responses, fmt.Sprintf("%d", info.id))
	}
	t.Logf("All %d watcher clients started and initial states verified", numWatchers)

	// --- 2. Trigger Register operation ---
	t.Logf("Triggering Register operation for path '%s'...", testPathName)
	var registerWg sync.WaitGroup
	registerWg.Add(1)
	go func() {
		defer registerWg.Done()
		fas.Register(testPathName)
	}()
	registerWg.Wait() // Ensure Register call itself completes before asserting responses
	t.Logf("Register operation for path '%s' completed", testPathName)

	// --- 3. Verify updates from all watchers ---
	t.Log("Verifying update responses from all watchers...")
	for _, info := range watchers {
		assertTargetUpdate(t, testCtx, info.responses, fmt.Sprintf("%d", info.id), testPathHostname)
	}

	// --- 4. Final Synchronization and Shutdown ---
	t.Log("Cancelling main test context to trigger service shutdown...")
	cancelTest() // This will also cancel the individual watcher contexts via testCtx.Done()

	t.Log("Waiting for all watcher goroutines to exit gracefully...")
	var watcherWg sync.WaitGroup
	watcherWg.Add(numWatchers)
	for _, info := range watchers {
		go func(doneCh <-chan struct{}, id int) {
			defer watcherWg.Done()
			select {
			case <-doneCh:
				t.Logf("Watcher %d goroutine exited gracefully after test cancellation.", id)
			case <-time.After(1 * time.Second): // A generous timeout for cleanup per watcher
				t.Errorf("Watcher %d goroutine did not exit gracefully after test cancellation. Potential resource leak.", id)
			}
		}(info.doneCh, info.id)
	}

	// Wait for all watcher goroutines to finish or timeout
	watcherWg.Wait()
	t.Log("All watcher goroutines accounted for")
}
