/*
Copyright (c) 2021 Nordix Foundation

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

package sqlite_test

import (
	"context"
	"os"
	"testing"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry/common"
	"github.com/nordix/meridio/pkg/nsp/registry/sqlite"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestTargetRegistrySQLite_Set(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	ctx := context.Background()

	nspTargets := []*nspAPI.Target{
		{
			Ips: []string{"172.16.0.1/32"},
			Context: map[string]string{
				"identifier": "32",
			},
			Status: nspAPI.Target_DISABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Ips: []string{"172.16.0.2/32"},
			Context: map[string]string{
				"identifier": "65",
			},
			Status: nspAPI.Target_DISABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Ips: []string{"172.16.0.3/32"},
			Context: map[string]string{
				"identifier": "1",
			},
			Status: nspAPI.Target_DISABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Conduit: &nspAPI.Conduit{
					Trench: &nspAPI.Trench{},
				},
			},
		},
	}

	// Add nspTargets[0]
	err = db.Set(ctx, nspTargets[0])
	assert.Nil(t, err)

	targets, err := db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, nspTargets[0], targets[0])

	// Update nspTargets[0]
	nspTargets[0].Status = nspAPI.Target_ENABLED
	err = db.Set(ctx, nspTargets[0])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, nspTargets[0], targets[0])

	// Add nspTargets[1]
	err = db.Set(ctx, nspTargets[1])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 2)
	assert.Contains(t, targets, nspTargets[0])
	assert.Contains(t, targets, nspTargets[1])

	// Add nspTargets[2]
	err = db.Set(ctx, nspTargets[2])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 3)
	assert.Contains(t, targets, nspTargets[0])
	assert.Contains(t, targets, nspTargets[1])
	assert.Contains(t, targets, nspTargets[2])

	// Update nspTargets[0] with different conduit
	// Due the way the data are stored, 2 objects (Stream, Conduit, Trench) with the same name causes problems.
	// A new nspTargets[0] entry will be created and old nspTargets[0] and nspTargets[1] will be deleted due to stream name conflict.
	nspTargets[0].Stream.Conduit.Name = "conduit-b"
	nspTargets[0].Context["identifier"] = "100"
	err = db.Set(ctx, nspTargets[0])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 2)
	assert.Contains(t, targets, nspTargets[0])
	assert.Contains(t, targets, nspTargets[2])

	// Remove nspTargets[0]
	err = db.Remove(ctx, nspTargets[0])
	assert.Nil(t, err)
	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 1)
	assert.Contains(t, targets, nspTargets[2])

	// Remove nspTargets[2]
	err = db.Remove(ctx, nspTargets[2])
	assert.Nil(t, err)
	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 0)
}

// drainNotification reports whether a notification is pending, without waiting.
// Waiting is not needed: Watch and Set push to the watcher's buffered channel
// (size 1, newer replaces older) before they return, so anything they sent is
// already there. An empty channel therefore means nothing was sent.
func drainNotification(w types.TargetWatcher) bool {
	select {
	case <-w.ResultChan():
		return true
	default:
		return false
	}
}

// TestTargetRegistrySQLite_Set_SkipsUnchanged verifies the no-op fast path in
// Set(): an update whose stored row already matches the incoming target must not
// re-broadcast to watchers, while a genuine change must.
func TestTargetRegistrySQLite_Set_SkipsUnchanged(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test_skip.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	ctx := context.Background()

	target := &nspAPI.Target{
		Ips:     []string{"172.16.0.1/32"},
		Context: map[string]string{"identifier": "32"},
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

	// Seed the target before watching so the watcher's initial snapshot is stable.
	assert.Nil(t, db.Set(ctx, target))

	watcher, err := db.Watch(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	defer watcher.Stop()

	// Watch queues an initial full snapshot; drain it so the channel is empty.
	assert.True(t, drainNotification(watcher), "expected initial snapshot on Watch")

	// Re-Set the identical target: the fast path must skip the fan-out entirely.
	assert.Nil(t, db.Set(ctx, target))
	assert.False(t, drainNotification(watcher),
		"unchanged Set must not notify watchers")

	// A real change (Context) must go through Save + notifyAllWatchers.
	target.Context["identifier"] = "99"
	assert.Nil(t, db.Set(ctx, target))
	assert.True(t, drainNotification(watcher),
		"changed Set must notify watchers")

	// A status change must also notify.
	target.Status = nspAPI.Target_DISABLED
	assert.Nil(t, db.Set(ctx, target))
	assert.True(t, drainNotification(watcher),
		"status change must notify watchers")

	// Another identical Set is skipped again.
	assert.Nil(t, db.Set(ctx, target))
	assert.False(t, drainNotification(watcher),
		"second unchanged Set must not notify watchers")
}

// TestTargetRegistrySQLite_Watch_StreamScopedSnapshot verifies the initial
// snapshot of a Watch when it is read only from the selected stream's rows.
//
// Before, Watch read the whole table and the watcher filtered it with its
// selector. Now, if the selector names a stream, Watch reads only that
// stream's rows, and the watcher still filters them. The risk is that the
// smaller read misses a target the selector would keep, so the watcher gets
// too few targets. The test compares each Watch snapshot with the old
// behavior: the full table, filtered with the same selector.
func TestTargetRegistrySQLite_Watch_StreamScopedSnapshot(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test_watch_snapshot.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	ctx := context.Background()
	stream := func(name string) *nspAPI.Stream {
		return &nspAPI.Stream{
			Name: name,
			Conduit: &nspAPI.Conduit{
				Name:   "conduit-a",
				Trench: &nspAPI.Trench{Name: "trench-a"},
			},
		}
	}

	// Step 1: store targets that the selectors below tell apart:
	//   - two in stream-a, enabled and disabled: both are read for stream-a,
	//     and only the watcher's filter removes the disabled one for an
	//     "enabled only" selector.
	//   - one in stream-b: must not be read for stream-a.
	//   - one frontend target without a stream (as frontends register): only
	//     reachable through the full-table path.
	for _, target := range []*nspAPI.Target{
		{Ips: []string{"172.16.0.1/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")},
		{Ips: []string{"172.16.0.2/32"}, Status: nspAPI.Target_DISABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")},
		{Ips: []string{"172.16.0.3/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-b")},
		{Ips: []string{"172.16.0.4/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_FRONTEND},
	} {
		assert.Nil(t, db.Set(ctx, target))
	}

	// snapshot opens a Watch, returns its initial target list and stops it.
	// No waiting is needed: Watch puts the initial list into the watcher's
	// channel before it returns.
	snapshot := func(t *testing.T, selector *nspAPI.Target) []*nspAPI.Target {
		t.Helper()
		w, err := db.Watch(ctx, selector)
		assert.Nil(t, err)
		defer w.Stop()
		select {
		case targets := <-w.ResultChan():
			return targets
		default:
			t.Fatal("expected initial snapshot on Watch")
			return nil
		}
	}
	// ids turns targets into "ID status" strings. The database does not
	// guarantee row order, so lists are compared as sets of these strings.
	ids := func(targets []*nspAPI.Target) []string {
		result := []string{}
		for _, target := range targets {
			result = append(result, sqlite.GetTargetID(target)+" "+target.GetStatus().String())
		}
		return result
	}

	// Step 2: build the reference. A Watch without selector names no stream,
	// so it reads the whole table, like every Watch did before.
	all := snapshot(t, nil)
	assert.Len(t, all, 4)

	// Step 3: one selector per case. want is the expected count, which guards
	// against the reference and the snapshot being wrong in the same way
	// (for example both empty).
	tests := []struct {
		name     string
		selector *nspAPI.Target
		want     int
	}{
		// Stream-scoped read: both stream-a targets.
		{"stream, any status", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")}, 2},
		// Stream-scoped read; the watcher's filter drops the disabled target.
		{"stream, enabled only", &nspAPI.Target{Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")}, 1},
		// Stream-scoped read of a different stream.
		{"other stream", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-b")}, 1},
		// Stream-scoped read that finds no rows.
		{"unknown stream", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-x")}, 0},
		// Stream-scoped read finds the stream-a rows; the watcher's filter drops
		// them because the conduit does not match. Checks that the watcher still
		// applies its full selector.
		{"stream, other conduit", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: &nspAPI.Stream{Name: "stream-a", Conduit: &nspAPI.Conduit{Name: "conduit-x"}}}, 0},
		// A stream without a name matches any stream: full-table read, all
		// default targets.
		{"stream without name", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: &nspAPI.Stream{}}, 3},
		// No stream at all, as the LB's frontend Watch uses: full-table read.
		{"no stream", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_FRONTEND}, 1},
	}

	// Step 4: for each selector, the snapshot must have the expected count and
	// equal the reference filtered with the same function the watcher uses.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snapshot(t, tt.selector)
			assert.Len(t, got, tt.want)
			assert.ElementsMatch(t, ids(common.Filter(tt.selector, all)), ids(got),
				"snapshot must equal the full list filtered by the selector")
		})
	}
}

// TestTargetRegistrySQLite_Broadcast_StreamGrouping verifies what watchers
// receive when a change is broadcast.
//
// A broadcast groups the targets by stream and gives a stream-scoped watcher
// only its stream's targets to filter; watchers without a stream get the full
// list. The risk is that a watcher gets a group that misses targets its
// selector would keep. The test opens watchers with different selectors,
// makes a change, and compares what each receives with the full list (from a
// watcher without selector) filtered with that watcher's selector.
func TestTargetRegistrySQLite_Broadcast_StreamGrouping(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test_broadcast_grouping.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	ctx := context.Background()
	stream := func(name string) *nspAPI.Stream {
		return &nspAPI.Stream{
			Name: name,
			Conduit: &nspAPI.Conduit{
				Name:   "conduit-a",
				Trench: &nspAPI.Trench{Name: "trench-a"},
			},
		}
	}

	// Step 1: store targets in two streams, plus a frontend target without a
	// stream (as frontends register).
	for _, target := range []*nspAPI.Target{
		{Ips: []string{"172.16.0.1/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")},
		{Ips: []string{"172.16.0.2/32"}, Status: nspAPI.Target_DISABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")},
		{Ips: []string{"172.16.0.3/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-b")},
		{Ips: []string{"172.16.0.4/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_FRONTEND},
	} {
		assert.Nil(t, db.Set(ctx, target))
	}

	// receive returns the pending target list of a watcher. No waiting is
	// needed: Watch and Set put the list into the watcher's channel before they
	// return.
	receive := func(t *testing.T, w types.TargetWatcher) []*nspAPI.Target {
		t.Helper()
		select {
		case targets := <-w.ResultChan():
			return targets
		default:
			t.Fatal("expected a target list")
			return nil
		}
	}
	// ids turns targets into "ID status" strings. The database does not
	// guarantee row order, so lists are compared as sets of these strings.
	ids := func(targets []*nspAPI.Target) []string {
		result := []string{}
		for _, target := range targets {
			result = append(result, sqlite.GetTargetID(target)+" "+target.GetStatus().String())
		}
		return result
	}

	// Step 2: open the watchers. The reference watcher has no selector, so a
	// broadcast gives it the full list. want is each watcher's expected count
	// after the change in step 3, which guards against the reference and the
	// received list being wrong in the same way.
	reference, err := db.Watch(ctx, nil)
	assert.Nil(t, err)
	defer reference.Stop()
	receive(t, reference) // drop the initial snapshot

	tests := []struct {
		name     string
		selector *nspAPI.Target
		want     int
	}{
		// stream-a group: the three stream-a targets after the change.
		{"stream, any status", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")}, 3},
		// stream-a group; the watcher's filter drops the disabled target.
		{"stream, enabled only", &nspAPI.Target{Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a")}, 2},
		// stream-b group: unaffected by the stream-a change, still delivered.
		{"other stream", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-b")}, 1},
		// No group for this stream: an empty list.
		{"unknown stream", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-x")}, 0},
		// stream-a group; the watcher's filter drops all of it because the
		// conduit does not match. Checks that the full selector still applies.
		{"stream, other conduit", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: &nspAPI.Stream{Name: "stream-a", Conduit: &nspAPI.Conduit{Name: "conduit-x"}}}, 0},
		// A stream without a name matches any stream: full list, all default
		// targets.
		{"stream without name", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT, Stream: &nspAPI.Stream{}}, 4},
		// No stream at all, as the LB's frontend Watch uses: full list.
		{"no stream", &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_FRONTEND}, 1},
	}
	watchers := make([]types.TargetWatcher, len(tests))
	for i, tt := range tests {
		w, err := db.Watch(ctx, tt.selector)
		assert.Nil(t, err)
		defer w.Stop()
		receive(t, w) // drop the initial snapshot, so only the broadcast is pending
		watchers[i] = w
	}

	// Step 3: a real change, a new enabled target in stream-a. It is not an
	// unchanged refresh, so Set broadcasts to all watchers.
	assert.Nil(t, db.Set(ctx, &nspAPI.Target{
		Ips: []string{"172.16.0.5/32"}, Status: nspAPI.Target_ENABLED, Type: nspAPI.Target_DEFAULT, Stream: stream("stream-a"),
	}))
	all := receive(t, reference)
	assert.Len(t, all, 5)

	// Step 4: every watcher got the broadcast, with the expected count, equal to
	// the full list filtered with the same function the watcher uses.
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := receive(t, watchers[i])
			assert.Len(t, got, tt.want)
			assert.ElementsMatch(t, ids(common.Filter(tt.selector, all)), ids(got),
				"broadcast must equal the full list filtered by the selector")
		})
	}
}

// TestTargetRegistrySQLite_Set_AbandonedCall verifies that a Set whose context
// is already done returns its error and writes nothing.
func TestTargetRegistrySQLite_Set_AbandonedCall(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test_set_abandoned.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	abandoned, cancel := context.WithCancel(context.Background())
	cancel()
	err = db.Set(abandoned, &nspAPI.Target{
		Ips:    []string{"172.16.0.1/32"},
		Status: nspAPI.Target_ENABLED,
		Type:   nspAPI.Target_DEFAULT,
	})
	assert.ErrorIs(t, err, context.Canceled)

	targets, err := db.Get(context.Background(), &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Empty(t, targets, "an abandoned Set must not write")
}

// TestTargetRegistrySQLite_Set_SkipsUnchangedWithoutStream verifies that the
// fast path also applies to targets without a stream, as frontends register:
// their ID and the one recomputed from the stored row must agree, otherwise
// every refresh would take the slow path and broadcast.
func TestTargetRegistrySQLite_Set_SkipsUnchangedWithoutStream(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test_skip_no_stream.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	ctx := context.Background()
	target := &nspAPI.Target{
		Ips:     []string{"frontend-a"},
		Context: map[string]string{"identifier": "frontend-a"},
		Type:    nspAPI.Target_FRONTEND,
	}
	assert.Nil(t, db.Set(ctx, target))

	watcher, err := db.Watch(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_FRONTEND})
	assert.Nil(t, err)
	defer watcher.Stop()
	assert.True(t, drainNotification(watcher), "expected initial snapshot on Watch")

	assert.Nil(t, db.Set(ctx, target))
	assert.False(t, drainNotification(watcher), "unchanged Set without stream must not notify watchers")
}
