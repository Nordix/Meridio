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

	"github.com/go-logr/logr"
	lbAPI "github.com/nordix/meridio/api/loadbalancer/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/stream"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testPathName                      = "test-path"
	testPathHostname                  = "test-path-hostname"
	mockWatchServerResponseBufferSize = 30
)

var forwardingAvailabilityTarget = &lbAPI.Target{Context: map[string]string{types.IdentifierKey: testPathHostname}}

// --- Test Helpers and Mocks ---

// mockStreamAvailabilityService_WatchServer is a test double for the gRPC WatchServer interface
type mockStreamAvailabilityService_WatchServer struct {
	lbAPI.StreamAvailabilityService_WatchServer
	ctx          context.Context
	responses    chan *lbAPI.Response // Channel to send responses back to the test goroutine
	sendDelay    time.Duration        // Simulate network latency for Send() calls
	sentMessages []*lbAPI.Response    // Slice to store history of successfully sent messages
	maxHistory   int                  // Maximum number of successfully sent messages to store in history
	mu           sync.Mutex
	logger       logr.Logger
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
		m.mu.Lock()
		// Store the message successfully sent
		clonedResp := proto.Clone(resp).(*lbAPI.Response)
		m.logger.V(1).Info("Send", "sent msg", clonedResp)
		m.sentMessages = append(m.sentMessages, clonedResp)
		// Sliding window to trim the history to maintain `maxHistory` size
		if len(m.sentMessages) > m.maxHistory {
			m.sentMessages = m.sentMessages[1:] // Get rid of the oldest element
		}
		m.mu.Unlock()
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

// getLastSentMessage provides the very last message sent by the mock server.
// Returns nil if no messages have been sent.
func (m *mockStreamAvailabilityService_WatchServer) getLastSentMessage() *lbAPI.Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sentMessages) == 0 {
		return nil
	}
	return m.sentMessages[len(m.sentMessages)-1]
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
	responses := make(chan *lbAPI.Response, mockWatchServerResponseBufferSize) // Buffered channel for responses
	mockServer := &mockStreamAvailabilityService_WatchServer{
		ctx:        ctx,
		responses:  responses,
		sendDelay:  sendDelay,
		maxHistory: 1, // As of now remember the last sent msg
	}
	mockServer.logger = log.Logger.WithName("mockStreamAvailabilityService_WatchServer").WithValues("maxHistory", mockServer.maxHistory)
	return mockServer, responses
}

// watcherClientInfo holds all relevant data for a single mock watcher client
type watcherClientInfo struct {
	id         string
	ctx        context.Context
	cancel     context.CancelFunc
	mockServer *mockStreamAvailabilityService_WatchServer
	responses  chan *lbAPI.Response
	doneCh     chan struct{} // Signals when the watcher goroutine has exited
}

// testEnvironment holds all shared components for a test case
type testEnvironment struct {
	testCtx    context.Context
	cancelTest context.CancelFunc
	fas        *stream.ForwardingAvailabilityService
	watchers   []*watcherClientInfo
}

// setupTestEnvironment initializes the ForwardingAvailabilityService and
// a specified number of watcher clients.
func setupTestEnvironment(
	t *testing.T,
	numWatchers int,
	testGetTargetDelay time.Duration,
	sendDelay time.Duration,
	testTimeout time.Duration,
) *testEnvironment {
	t.Helper()

	testCtx, cancelTest := context.WithTimeout(context.Background(), testTimeout)
	// Initialize logger
	_ = log.New("test", "INFO") // TODO: make this part of a global test setup method

	fas := stream.NewForwardingAvailabilityServiceForTest(testCtx, forwardingAvailabilityTarget, testGetTargetDelay)
	require.NotNil(t, fas, "ForwardingAvailabilityService should be created")

	watchers := make([]*watcherClientInfo, numWatchers)
	for i := 0; i < numWatchers; i++ {
		watcherID := fmt.Sprintf("%v", i+1)
		watcherClientCtx, watcherCancel := context.WithCancel(testCtx)
		mockWatcherServer, responses := newMockWatcherServer(watcherClientCtx, sendDelay)
		doneCh := make(chan struct{})

		info := &watcherClientInfo{
			id:         watcherID,
			ctx:        watcherClientCtx,
			cancel:     watcherCancel,
			mockServer: mockWatcherServer,
			responses:  responses,
			doneCh:     doneCh,
		}
		watchers[i] = info

		go func(info *watcherClientInfo) {
			defer close(info.doneCh)
			err := fas.Watch(&emptypb.Empty{}, info.mockServer)
			if err != nil && err != info.ctx.Err() && err != testCtx.Err() {
				t.Errorf("Watcher %v exited with unexpected error: %v", info.id, err)
			}
		}(info)

		assertEmptyTargetUpdate(t, testCtx, info.responses, info.id)
		t.Logf("Watcher %v client started and initial state verified", info.id)
	}
	t.Logf("All %d watcher clients started and initial states verified", numWatchers)

	return &testEnvironment{
		testCtx:    testCtx,
		cancelTest: cancelTest,
		fas:        fas,
		watchers:   watchers,
	}
}

// tearDownTestEnvironment performs common cleanup for test environment
func (te *testEnvironment) tearDownTestEnvironment(t *testing.T) {
	t.Helper()

	t.Log("Cancelling main test context to trigger service shutdown...")
	te.cancelTest() // This will also cancel individual watcher contexts

	t.Log("Waiting for all watcher goroutines to exit gracefully...")
	var watcherWg sync.WaitGroup
	watcherWg.Add(len(te.watchers))
	for _, info := range te.watchers {
		go func(doneCh <-chan struct{}, id string) {
			defer watcherWg.Done()
			select {
			case <-doneCh:
				t.Logf("Watcher %v goroutine exited gracefully after test cancellation", id)
			case <-time.After(1 * time.Second): // Generous timeout for individual watcher cleanup
				t.Errorf("Watcher %v goroutine did not exit gracefully after test cancellation. Potential resource leak.", id)
			}
		}(info.doneCh, info.id)
	}
	watcherWg.Wait()
	t.Log("All watcher goroutines accounted for")
}

// waitForGoroutineCompletion executes a function in a goroutine and waits for its
// completion, or a timeout, reporting an error if the timeout is hit.
func waitForGoroutineCompletion(t *testing.T, ctx context.Context, op func(), msg string, operationTimeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		op()
	}()

	select {
	case <-done:
		t.Logf("%s completed", msg)
	case <-ctx.Done(): // Use testCtx for overall test timeout
		t.Fatalf("Test context cancelled before %s completed: %v", msg, ctx.Err())
	case <-time.After(operationTimeout): // Specific operation timeout
		t.Fatalf("Timeout waiting for %s", msg)
	}
}

// assertEmptyTargetUpdate waits for and verifies an update with an empty target
func assertEmptyTargetUpdate(t *testing.T, ctx context.Context, responses <-chan *lbAPI.Response, watcherID string) {
	t.Helper() // Mark this as a test helper

	select {
	case resp, ok := <-responses:
		assert.True(t, ok, "Watcher %s should receive empty target response", watcherID)
		require.NotNil(t, resp, "Watcher %s empty target response should not be nil", watcherID)
		require.Len(t, resp.GetTargets(), 1, "Watcher %s empty target response should contain one target", watcherID)
		assert.True(t, len(resp.GetTargets()[0].GetContext()) == 0, "Watcher %s empty target response context should be empty (actual: %v)", watcherID, resp.GetTargets()[0].GetContext())
		t.Logf("Watcher %s received empty target update successfully", watcherID)
	case <-ctx.Done():
		t.Fatalf("Test context cancelled before watcher %s empty target response received: %v", watcherID, ctx.Err())
	case <-time.After(1 * time.Second): // A generous timeout for updates to propagate
		t.Fatalf("Timeout waiting for watcher %s empty target response", watcherID)
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

func Test_Register_Unregister(t *testing.T) {
	testGetTargetDelay := 0 * time.Millisecond // No delay in fas.getTarget()
	sendDelay := 5 * time.Millisecond          // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second             // Overall test timeout

	te := setupTestEnvironment(t, 1, testGetTargetDelay, sendDelay, testTimeout)
	defer te.tearDownTestEnvironment(t)

	watcherInfo := te.watchers[0] // Get the single watcher info

	// --- Step 1: Trigger Register operation ---
	t.Logf("Triggering Register operation for path '%s'...", testPathName)
	waitForGoroutineCompletion(t, te.testCtx, func() { te.fas.Register(testPathName) }, "Register goroutine", time.Second)

	t.Log("Verifying update response for Register from watcher...")
	assertTargetUpdate(t, te.testCtx, watcherInfo.responses, watcherInfo.id, testPathHostname)

	// --- Step 2: Trigger Unregister operation ---
	t.Logf("Triggering Unregister operation for path '%s'...", testPathName)
	waitForGoroutineCompletion(t, te.testCtx, func() { te.fas.Unregister(testPathName) }, "Unregister goroutine", time.Second)

	t.Log("Verifying update response for Unregister from watcher...")
	assertEmptyTargetUpdate(t, te.testCtx, watcherInfo.responses, watcherInfo.id)
}

func Test_RegisterMultiplePaths_And_StopService(t *testing.T) {
	testGetTargetDelay := 0 * time.Millisecond // No delay in fas.getTarget()
	sendDelay := 5 * time.Millisecond          // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second             // Overall test timeout
	numOfTestPaths := 4

	te := setupTestEnvironment(t, 1, testGetTargetDelay, sendDelay, testTimeout)
	defer te.tearDownTestEnvironment(t)

	watcherInfo := te.watchers[0] // Get the single watcher info

	// --- Step 1: Trigger Register operation for each availability path ---
	triggerRegisterOperations := func() {
		var registerWg sync.WaitGroup
		registerWg.Add(numOfTestPaths)
		for i := range numOfTestPaths {
			go func(pathName string) {
				defer registerWg.Done()
				t.Logf("Triggering Register operation for path '%s'...", pathName)
				te.fas.Register(pathName)
			}(fmt.Sprintf("%v-%v", testPathName, i))
		}
		registerWg.Wait()
	}
	waitForGoroutineCompletion(t, te.testCtx, triggerRegisterOperations, "Register operations goroutine", time.Second)

	t.Log("Verifying update response for Register from watcher...")
	assertTargetUpdate(t, te.testCtx, watcherInfo.responses, watcherInfo.id, testPathHostname)

	// --- 2. Synchronize via fas.Stop() and verify final shutdown state ---
	// This will notify watchers and wait for them to process/send. Once fas.Stop()
	// concluded in time, we can be certain that there should be no lingering updates
	// and watchers are done processing everything i.e. they are not stuck.
	// Note: fas.Stop() has a built in 2 Seconds timeout
	t.Log("Calling fas.Stop() to gracefully shut down and synchronize watchers...")
	waitForGoroutineCompletion(t, te.testCtx, func() { te.fas.Stop() }, "fas.Stop()", 1*time.Second)

	// After Stop() returns, the very last message in the send history of the mock server
	// must be the unavailable (empty) target.
	expectedUnavailableTarget := &lbAPI.Target{}
	lastSent := watcherInfo.mockServer.getLastSentMessage()
	t.Logf("Verifying last sent msg (%v) of the Mock server is the unavailable (empty) target...", lastSent)
	assert.True(t, proto.Equal(lastSent, &lbAPI.Response{Targets: []*lbAPI.Target{expectedUnavailableTarget}}),
		"Last sent message (%+v) should be the unavailable target (%+v)", lastSent, expectedUnavailableTarget)

	// Somewhat redundant after checking the last sent msg, but verify empty update response after Stop()
	// that is expected because of previous Register operations.
	t.Log("Verifying empty update response after Stop()...")
	assertEmptyTargetUpdate(t, te.testCtx, watcherInfo.responses, watcherInfo.id)
}

func Test_RapidRegisterUnregister_StressAndDeduplication(t *testing.T) {
	testGetTargetDelay := 1 * time.Millisecond // Minimal delay in fas.getTarget() to increase contention
	sendDelay := 5 * time.Millisecond          // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second             // Overall test timeout
	numOfTestPaths := 2
	numOfOperationSequences := 5

	te := setupTestEnvironment(t, 1, testGetTargetDelay, sendDelay, testTimeout)
	defer te.tearDownTestEnvironment(t)

	watcherInfo := te.watchers[0] // Get the single watcher info

	// --- Step 1: Trigger combined Register-Unregister operations for each availability path multiple times ---
	// Note: mind mockWatchServerResponseBufferSize before increasing numOfTestPaths, numOfOperationSequences (or add a response drainer)
	triggerRegisterOperations := func() {
		var opWg sync.WaitGroup
		opWg.Add(numOfOperationSequences * numOfTestPaths)
		for i := range numOfOperationSequences {
			for j := range numOfTestPaths {
				go func(pathName string, sequence int) {
					defer opWg.Done()
					t.Logf("Triggering Register and Unregister operations (seq %d) for path '%s'...", sequence, pathName)
					te.fas.Register(pathName)
					te.fas.Unregister(pathName)
				}(fmt.Sprintf("%v-%v", testPathName, j), i)
			}
		}
		opWg.Wait()
	}
	waitForGoroutineCompletion(t, te.testCtx, triggerRegisterOperations, "Register-Unregister operations goroutine", 2*time.Second)

	// --- 2. Synchronize via fas.Stop() and verify final shutdown state ---
	// This will notify watchers and wait for them to process/send. Once fas.Stop()
	// concluded in time, we can be certain that there should be no lingering updates
	// and watchers are done processing everything i.e. they are not stuck.
	// Note: fas.Stop() has a built in 2 Seconds timeout
	t.Log("Calling fas.Stop() to gracefully shut down and synchronize watchers...")
	waitForGoroutineCompletion(t, te.testCtx, func() { te.fas.Stop() }, "fas.Stop()", 1*time.Second)

	// After Stop() returns, the very last message in the send history of the mock server
	// must be the unavailable (empty) target.
	expectedUnavailableTarget := &lbAPI.Target{}
	lastSent := watcherInfo.mockServer.getLastSentMessage()
	t.Logf("Verifying last sent msg (%v) of the Mock server is the unavailable (empty) target...", lastSent)
	assert.True(t, proto.Equal(lastSent, &lbAPI.Response{Targets: []*lbAPI.Target{expectedUnavailableTarget}}),
		"Last sent message (%+v) should be the unavailable target (%+v)", lastSent, expectedUnavailableTarget)
}

func Test_DeadlockScenario_WithDelayedGetTarget(t *testing.T) {
	testGetTargetDelay := 50 * time.Millisecond // Simulate delay in getTarget() for contention
	sendDelay := 5 * time.Millisecond           // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second              // Overall test timeout

	te := setupTestEnvironment(t, 1, testGetTargetDelay, sendDelay, testTimeout)
	defer te.tearDownTestEnvironment(t)

	watcherInfo := te.watchers[0]

	// --- Step 1: Trigger concurrent Register/Unregister operations ---
	// Unregister routine waits for Register to conclude to secure wanted
	// execution order. Aim is to provoke contention for `fas.mu.Lock()`
	// while the watcher is potentially delayed in its `getTarget()` call.
	t.Logf("Triggering concurrent Register/Unregister operations for path '%s'...", testPathName)
	registerDone := make(chan struct{})
	unregisterDone := make(chan struct{})
	var opsWg sync.WaitGroup
	opsWg.Add(2)

	go func() {
		defer func() {
			close(registerDone)
			opsWg.Done()
		}()
		te.fas.Register(testPathName)
	}()

	go func() {
		defer func() {
			close(unregisterDone)
			opsWg.Done()
		}()
		<-registerDone // Ensure Register completes first
		te.fas.Unregister(testPathName)
	}()

	waitForGoroutineCompletion(t, te.testCtx, func() { opsWg.Wait() }, "Register/Unregister goroutines", 1*time.Second)
	t.Log("All Register/Unregister goroutines completed (no deadlock detected in ops)")

	// --- 2. Synchronize via fas.Stop() and verify final shutdown state ---
	// This will notify watchers and wait for them to process/send. Once fas.Stop()
	// concluded in time, we can be certain that there should be no lingering updates
	// and watchers are done processing everything i.e. they are not stuck.
	// Note: fas.Stop() has a built in 2 Seconds timeout
	t.Log("Calling fas.Stop() to gracefully shut down and synchronize watchers...")
	waitForGoroutineCompletion(t, te.testCtx, func() { te.fas.Stop() }, "fas.Stop()", 1*time.Second)

	// After Stop() returns, the very last message in the send history of the mock server
	// must be the unavailable (empty) target.
	expectedUnavailableTarget := &lbAPI.Target{}
	lastSent := watcherInfo.mockServer.getLastSentMessage()
	t.Logf("Verifying last sent msg (%v) of the Mock server is the unavailable (empty) target...", lastSent)
	assert.True(t, proto.Equal(lastSent, &lbAPI.Response{Targets: []*lbAPI.Target{expectedUnavailableTarget}}),
		"Last sent message (%+v) should be the unavailable target (%+v)", lastSent, expectedUnavailableTarget)
}

func Test_Register_WithMultipleWatchers(t *testing.T) {
	const numWatchers = 2 // watchers to test with

	testGetTargetDelay := 0 * time.Millisecond // No delay in fas.getTarget()
	sendDelay := 5 * time.Millisecond          // Simulate latency in gRPC Send
	testTimeout := 5 * time.Second             // Overall test timeout

	te := setupTestEnvironment(t, numWatchers, testGetTargetDelay, sendDelay, testTimeout)
	defer te.tearDownTestEnvironment(t)

	// --- 1. Trigger Register operation ---
	t.Logf("Triggering Register operation for path '%s'...", testPathName)
	waitForGoroutineCompletion(t, te.testCtx, func() { te.fas.Register(testPathName) }, "Register operation", 1*time.Second)

	// --- 2. Verify updates from all watchers ---
	t.Log("Verifying update responses from all watchers...")
	for _, info := range te.watchers {
		assertTargetUpdate(t, te.testCtx, info.responses, info.id, testPathHostname)
	}
}
