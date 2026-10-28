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

package ctxlock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statsTestTimeout bounds acquires that should succeed and the wait for a
// goroutine to be queued, so a broken invariant fails instead of hanging.
const statsTestTimeout = 2 * time.Second

// lockInBackground starts a Lock call with ctx in a goroutine, waits until the
// call is queued on the lock, and returns a channel with its result.
func lockInBackground(t *testing.T, ctx context.Context, m *Mutex, s *LockStats) <-chan error {
	t.Helper()
	before := s.waiters.Load()
	result := make(chan error, 1)
	go func() { result <- m.Lock(ctx) }()
	require.Eventually(t, func() bool { return s.waiters.Load() == before+1 },
		statsTestTimeout, time.Millisecond, "Lock call not queued")
	return result
}

func receive(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(statsTestTimeout):
		t.Fatal("Lock call did not return")
		return nil
	}
}

// Test_Stats_Uncontended verifies that acquisitions without waiting are counted
// as acquired only.
func Test_Stats_Uncontended(t *testing.T) {
	var sw StatsSwitch
	s := sw.Enable()
	m := NewWithStats(&sw)
	for range 3 {
		require.NoError(t, m.Lock(context.Background()))
		m.Unlock()
	}

	assert.Equal(t, LockSnapshot{Acquired: 3}, s.Snapshot())
}

// Test_Stats_DoneOnEntry verifies that a Lock call with an already done context
// is counted as abandoned.
func Test_Stats_DoneOnEntry(t *testing.T) {
	var sw StatsSwitch
	s := sw.Enable()
	m := NewWithStats(&sw)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, m.Lock(ctx), context.Canceled)

	snap := s.Snapshot()
	assert.Equal(t, int64(1), snap.Abandoned)
	assert.Equal(t, int64(0), snap.Acquired)
	assert.Equal(t, int64(0), snap.Waiters)
}

// Test_Stats_Contended verifies the counters and peaks of a wait that ends with
// the lock acquired, and that Snapshot starts a new peak period.
func Test_Stats_Contended(t *testing.T) {
	var sw StatsSwitch
	s := sw.Enable()
	m := NewWithStats(&sw)
	require.NoError(t, m.Lock(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), statsTestTimeout)
	defer cancel()
	result := lockInBackground(t, ctx, m, s)
	m.Unlock()
	require.NoError(t, receive(t, result))
	m.Unlock()

	snap := s.Snapshot()
	assert.Equal(t, int64(2), snap.Acquired)
	assert.Equal(t, int64(1), snap.Contended)
	assert.Equal(t, int64(0), snap.Abandoned)
	assert.Equal(t, int64(0), snap.Waiters)
	assert.Equal(t, int64(1), snap.PeakWaiters)
	assert.Positive(t, snap.TotalWait)
	assert.Equal(t, snap.TotalWait, snap.PeakWait, "single wait")

	next := s.Snapshot()
	assert.Equal(t, int64(0), next.PeakWaiters)
	assert.Equal(t, time.Duration(0), next.PeakWait)
	assert.Equal(t, snap.Acquired, next.Acquired)
	assert.Equal(t, snap.TotalWait, next.TotalWait)
}

// Test_Stats_AbandonedWhileWaiting verifies that a wait ended by cancellation
// is counted as abandoned and its wait time is included.
func Test_Stats_AbandonedWhileWaiting(t *testing.T) {
	var sw StatsSwitch
	s := sw.Enable()
	m := NewWithStats(&sw)
	require.NoError(t, m.Lock(context.Background()))
	defer m.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	result := lockInBackground(t, ctx, m, s)
	cancel()
	require.ErrorIs(t, receive(t, result), context.Canceled)

	snap := s.Snapshot()
	assert.Equal(t, int64(1), snap.Acquired, "only the holder")
	assert.Equal(t, int64(0), snap.Contended)
	assert.Equal(t, int64(1), snap.Abandoned)
	assert.Equal(t, int64(0), snap.Waiters)
	assert.Equal(t, int64(1), snap.PeakWaiters)
	assert.Positive(t, snap.TotalWait)
}

// Test_Stats_Concurrent verifies the counter invariants under concurrent use.
func Test_Stats_Concurrent(t *testing.T) {
	const n = 50
	var sw StatsSwitch
	s := sw.Enable()
	m := NewWithStats(&sw)
	ctx, cancel := context.WithTimeout(context.Background(), statsTestTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			if assert.NoError(t, m.Lock(ctx)) {
				m.Unlock()
			}
		})
	}
	wg.Wait()

	snap := s.Snapshot()
	assert.Equal(t, int64(n), snap.Acquired)
	assert.Equal(t, int64(0), snap.Abandoned)
	assert.Equal(t, int64(0), snap.Waiters)
	assert.LessOrEqual(t, snap.Contended, snap.Acquired)
	assert.LessOrEqual(t, snap.PeakWaiters, int64(n))
}

// Test_Stats_Disabled verifies that nothing is recorded while the switch is
// disabled and that Enable starts from zero.
func Test_Stats_Disabled(t *testing.T) {
	var sw StatsSwitch
	m := NewWithStats(&sw)
	require.Nil(t, sw.Stats())
	require.NoError(t, m.Lock(context.Background()))
	m.Unlock()

	s := sw.Enable()
	require.NoError(t, m.Lock(context.Background()))
	m.Unlock()
	sw.Disable()
	require.NoError(t, m.Lock(context.Background()))
	m.Unlock()

	assert.Equal(t, LockSnapshot{Acquired: 1}, s.Snapshot(), "only the call while enabled")
	assert.Equal(t, LockSnapshot{}, sw.Enable().Snapshot(), "re-enable starts from zero")
}

// Test_Stats_SwitchDuringWait verifies that a wait in progress completes its
// accounting on the LockStats installed when it started, so switching stats
// off and on again leaves both the old and the new LockStats consistent.
func Test_Stats_SwitchDuringWait(t *testing.T) {
	var sw StatsSwitch
	old := sw.Enable()
	m := NewWithStats(&sw)
	require.NoError(t, m.Lock(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), statsTestTimeout)
	defer cancel()
	result := lockInBackground(t, ctx, m, old)
	sw.Disable()
	fresh := sw.Enable()
	m.Unlock()
	require.NoError(t, receive(t, result))
	m.Unlock()

	oldSnap := old.Snapshot()
	assert.Equal(t, int64(2), oldSnap.Acquired)
	assert.Equal(t, int64(1), oldSnap.Contended)
	assert.Equal(t, int64(0), oldSnap.Waiters)
	assert.Equal(t, LockSnapshot{}, fresh.Snapshot())
}

// Test_Stats_EnableDuringWait verifies that Enable during a wait leaves the
// wait accounted on the LockStats it started with, and that the new LockStats
// starts empty.
func Test_Stats_EnableDuringWait(t *testing.T) {
	var sw StatsSwitch
	old := sw.Enable()
	m := NewWithStats(&sw)
	require.NoError(t, m.Lock(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), statsTestTimeout)
	defer cancel()
	result := lockInBackground(t, ctx, m, old)
	// Enable while enabled replaces old with a new LockStats. The queued call
	// keeps recording into old.
	fresh := sw.Enable()
	m.Unlock()
	require.NoError(t, receive(t, result))
	m.Unlock()

	oldSnap := old.Snapshot()
	assert.Equal(t, int64(2), oldSnap.Acquired)
	assert.Equal(t, int64(1), oldSnap.Contended)
	assert.Equal(t, int64(0), oldSnap.Waiters)
	assert.Equal(t, int64(1), oldSnap.PeakWaiters)
	assert.Positive(t, oldSnap.TotalWait)
	assert.Equal(t, LockSnapshot{}, fresh.Snapshot())
}
