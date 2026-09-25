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

// White-box tests (package sqlite, not sqlite_test) for behavior that depends on
// unexported state — here the notifyDirty flag. A real getAll failure cannot be
// injected from a black-box test without a production seam (dropping the streams
// table breaks the write via FullSaveAssociations rather than isolating the read,
// verified by probing), so we set notifyDirty directly to model the post-failure
// state and assert the fast-path recovery logic that the flag guards.
package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func drainWatcher(w types.TargetWatcher) bool {
	select {
	case <-w.ResultChan():
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

// TestSet_ReNotifiesWhenDirty verifies that when notifyDirty is set (a prior
// broadcast failed, leaving watchers behind the DB), the next unchanged refresh
// does NOT take the no-op fast path: it re-broadcasts and clears the flag. Once
// clean, a further unchanged refresh skips notify again.
//
// This is the regression guard for the divergence bug: without the notifyDirty
// check, a Set whose notifyAllWatchers failed after a successful write would be
// permanently masked, because every later identical refresh compares equal and
// returns early.
func TestSet_ReNotifiesWhenDirty(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test_dirty_internal.db"
	os.Remove(dbFile)
	trsql, err := New(dbFile)
	assert.Nil(t, err)
	defer func() {
		trsql.Close()
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

	// Seed the target with a clean broadcast, then register a watcher and drain
	// its initial snapshot so the channel starts empty and notifyDirty is false.
	assert.Nil(t, trsql.Set(ctx, target))
	watcher, err := trsql.Watch(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	defer watcher.Stop()
	assert.True(t, drainWatcher(watcher), "expected initial snapshot on Watch")
	assert.False(t, trsql.notifyDirty, "flag should be clean after a successful broadcast")

	// Model the post-failure state: a previous notify failed, watchers are behind.
	trsql.notifyDirty = true

	// Unchanged refresh: must re-broadcast (not skip) and clear the flag.
	assert.Nil(t, trsql.Set(ctx, target))
	assert.True(t, drainWatcher(watcher),
		"unchanged refresh while dirty must re-notify")
	assert.False(t, trsql.notifyDirty, "successful re-broadcast must clear the flag")

	// Debt paid: a further identical refresh takes the fast path and skips notify.
	assert.Nil(t, trsql.Set(ctx, target))
	assert.False(t, drainWatcher(watcher),
		"unchanged refresh while clean must skip notify")
}
