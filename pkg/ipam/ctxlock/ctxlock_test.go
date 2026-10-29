/*
Copyright (c) 2026 OpenInfra Foundation Europe

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

package ctxlock_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/ipam/ctxlock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// acquireTimeout bounds every "should succeed" acquire in these tests so that a
// violated invariant (the lock unexpectedly held) fails with a clear timeout
// instead of hanging until the whole test binary's deadline.
const acquireTimeout = 2 * time.Second

// Test_LockUnlock_Uncontended verifies a basic acquire/release cycle and that
// the lock can be re-acquired afterwards.
func Test_LockUnlock_Uncontended(t *testing.T) {
	m := ctxlock.New()

	require.NoError(t, m.Lock(context.Background()))
	m.Unlock()

	// Re-acquire to confirm Unlock made it available again.
	require.NoError(t, m.Lock(context.Background()))
	m.Unlock()
}

// Test_MutualExclusion verifies that a second Lock does not acquire while the
// lock is held, and succeeds once it is released.
//
// The "still held" assertion uses a non-blocking observation (the acquired
// channel is not yet closed), which is a sound, monotonic check: it proves the
// waiter has NOT acquired. It deliberately does not try to prove the waiter has
// reached a blocked state, which is not observable without racy timing.
func Test_MutualExclusion(t *testing.T) {
	m := ctxlock.New()
	require.NoError(t, m.Lock(context.Background()))

	acquired := make(chan struct{})
	go func() {
		_ = m.Lock(context.Background())
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second Lock acquired while the lock was held")
	case <-time.After(100 * time.Millisecond):
		// Expected: has not acquired while held.
	}

	m.Unlock()

	select {
	case <-acquired:
		// Expected: second Lock proceeded after release.
	case <-time.After(acquireTimeout):
		t.Fatal("second Lock did not acquire after release")
	}
	m.Unlock()
}

// Test_Lock_AlreadyCancelledCtx verifies the fast path: when ctx is already
// done, Lock returns the ctx error and does not acquire, even if the lock is
// free.
func Test_Lock_AlreadyCancelledCtx(t *testing.T) {
	m := ctxlock.New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.Lock(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// The lock must still be free (the cancelled Lock must not have acquired).
	// Bounded context so a violated invariant fails via timeout, not a hang.
	ctx2, cancel2 := context.WithTimeout(context.Background(), acquireTimeout)
	defer cancel2()
	require.NoError(t, m.Lock(ctx2), "lock should be free after an already-cancelled Lock")
	m.Unlock()
}

// Test_Lock_DeadlineExceeded deterministically exercises the cancel-while-
// waiting path: the lock is held for the whole test, so a waiter with a short
// timeout is guaranteed to still be waiting when its deadline fires, forcing it
// through the "context done while blocked" branch. No sleep-based
// synchronization is needed; the timeout fires regardless of scheduling.
func Test_Lock_DeadlineExceeded(t *testing.T) {
	m := ctxlock.New()
	require.NoError(t, m.Lock(context.Background())) // hold it for the whole test
	defer m.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := m.Lock(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// Test_Lock_CancelledWaiterDoesNotHoldLock verifies the invariant that matters
// for a cancelled waiter: it returns an error and does NOT end up holding the
// lock, so it never blocks later acquirers. This holds whether the cancellation
// hit the fast path or the waiting path, so the test needs no timing hack to
// force a particular path.
func Test_Lock_CancelledWaiterDoesNotHoldLock(t *testing.T) {
	m := ctxlock.New()
	require.NoError(t, m.Lock(context.Background())) // hold it

	ctx, cancel := context.WithCancel(context.Background())
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- m.Lock(ctx)
	}()

	cancel() // cancel the waiter (may be before or after it parks)

	select {
	case err := <-waitErr:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(acquireTimeout):
		t.Fatal("cancelled waiter did not return")
	}

	// Invariant: the cancelled waiter holds nothing. Releasing the original
	// holder must hand the lock to a fresh caller. Bounded context so a
	// violated invariant fails via timeout instead of hanging forever.
	m.Unlock()
	ctx2, cancel2 := context.WithTimeout(context.Background(), acquireTimeout)
	defer cancel2()
	require.NoError(t, m.Lock(ctx2), "lock should be free after a cancelled waiter")
	m.Unlock()
}

// Test_Unlock_NotHeld_Panics verifies that releasing an unheld lock panics,
// guarding against unbalanced Unlock calls.
func Test_Unlock_NotHeld_Panics(t *testing.T) {
	m := ctxlock.New()
	assert.Panics(t, func() { m.Unlock() })
}

// Test_Concurrent_NoDoubleHolder stress-checks mutual exclusion: many
// goroutines repeatedly lock, increment a shared counter without atomics, and
// unlock. A data race or double-holder would corrupt the count (and trip the
// race detector under `go test -race`).
func Test_Concurrent_NoDoubleHolder(t *testing.T) {
	m := ctxlock.New()
	const goroutines = 50
	const iterations = 200

	var counter int
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := m.Lock(context.Background()); err != nil {
					t.Errorf("unexpected Lock error: %v", err)
					return
				}
				counter++ // protected by the lock
				m.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, goroutines*iterations, counter)
}
