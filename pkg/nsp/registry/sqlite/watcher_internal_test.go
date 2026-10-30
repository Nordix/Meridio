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

// White-box tests for watcher removal; they inspect the unexported watcher list.
package sqlite

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func newWatcherTestRegistry(t *testing.T, dbFile string) *TargetRegistrySQLite {
	t.Helper()
	os.Remove(dbFile)
	trsql, err := New(dbFile)
	assert.Nil(t, err)
	t.Cleanup(func() {
		trsql.Close()
		os.Remove(dbFile)
	})
	return trsql
}

func watcherTestTarget(ip string, identifier string) *nspAPI.Target {
	return &nspAPI.Target{
		Ips:     []string{ip},
		Context: map[string]string{"identifier": identifier},
		Status:  nspAPI.Target_ENABLED,
		Type:    nspAPI.Target_DEFAULT,
		Stream: &nspAPI.Stream{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name:   "conduit-a",
				Trench: &nspAPI.Trench{Name: "trench-a"},
			},
		},
	}
}

func watcherCount(trsql *TargetRegistrySQLite) int {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	return len(trsql.watchers)
}

// TestWatch_StopRemovesWatcher models TAPA refreshes: an unchanged Register
// followed by a short verify Watch. Unchanged refreshes do not broadcast, so
// Stop itself must remove the watcher, or the watcher list grows forever.
func TestWatch_StopRemovesWatcher(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	trsql := newWatcherTestRegistry(t, "test_watcher_stop_internal.db")
	ctx := context.Background()
	target := watcherTestTarget("172.16.0.1/32", "1")
	assert.Nil(t, trsql.Set(ctx, target))

	// A long-lived watcher (like an LB) must stay registered.
	live, err := trsql.Watch(ctx, nil)
	assert.Nil(t, err)
	assert.True(t, drainWatcher(live), "expected initial snapshot on Watch")

	selector := &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: target.Stream}
	for range 100 {
		assert.Nil(t, trsql.Set(ctx, target)) // unchanged: fast path, no broadcast
		w, err := trsql.Watch(ctx, selector)
		assert.Nil(t, err)
		w.Stop()
	}
	assert.Equal(t, 1, watcherCount(trsql), "only the live watcher may remain")

	// The live watcher still receives changes.
	assert.Nil(t, trsql.Set(ctx, watcherTestTarget("172.16.0.2/32", "2")))
	assert.True(t, drainWatcher(live), "live watcher must be notified")

	live.Stop()
	assert.Equal(t, 0, watcherCount(trsql))
}

// TestWatch_ConcurrentStopAndBroadcast checks that watchers stopping while a
// broadcast is running is safe. The fix makes Stop take the registry lock to
// remove the watcher, while a broadcast holds the registry lock and takes each
// watcher's lock. The test runs both concurrently and checks for:
//   - deadlock: Stop and a broadcast waiting on each other's lock forever.
//     Caught by the timeout below.
//   - data races on the watcher list or the stopped flag. Caught only when run
//     with -race.
//   - wrong cleanup: stopped watchers left in the list, or live ones removed.
//     Caught by the watcher counts at the end.
func TestWatch_ConcurrentStopAndBroadcast(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	trsql := newWatcherTestRegistry(t, "test_watcher_concurrent_internal.db")
	ctx := context.Background()

	// Step 1: long-lived watchers, like LB and frontend streams. Each gets a
	// reader goroutine that drains its mailbox, like Server.watcher does, so
	// broadcasts keep delivering to them during the test. A reader exits when
	// Stop closes the watcher's channel.
	const liveCount = 2
	var readers sync.WaitGroup
	var lives []types.TargetWatcher
	for range liveCount {
		w, err := trsql.Watch(ctx, nil)
		assert.Nil(t, err)
		lives = append(lives, w)
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range w.ResultChan() {
			}
		}()
	}

	var wg sync.WaitGroup

	// Step 2: broadcasters. Each Set changes the target's identifier, so it
	// never takes the fast path: every call broadcasts, taking the registry
	// lock and then each watcher's lock.
	for g := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 50 {
				target := watcherTestTarget(fmt.Sprintf("172.16.%d.1/32", g), fmt.Sprint(i))
				if err := trsql.Set(ctx, target); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	// Step 3: short Watches, like TAPA's verify Watch on each refresh: open,
	// read the snapshot, Stop. Stop takes the watcher lock, then the registry
	// lock. Running alongside step 2, a Stop regularly hits a watcher that a
	// broadcast is notifying, which is the overlap the test is after.
	selector := &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				w, err := trsql.Watch(ctx, selector)
				if err != nil {
					t.Error(err)
					return
				}
				drainWatcher(w)
				w.Stop()
			}
		}()
	}

	// Step 4: wait for steps 2 and 3. Normally this takes well under a second,
	// so not finishing within the timeout means a deadlock.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("Set and Watch/Stop did not finish: deadlock suspected")
	}

	// Step 5: every short watcher removed itself; the live ones are untouched.
	assert.Equal(t, liveCount, watcherCount(trsql), "only the live watchers may remain")

	// Step 6: stopping the live watchers empties the list and ends their
	// readers (goleak checks that no goroutine is left behind).
	for _, w := range lives {
		w.Stop()
	}
	readers.Wait()
	assert.Equal(t, 0, watcherCount(trsql))
}
