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

package sqlite_test

import (
	"context"
	"os"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The store lock is context-aware: a store operation invoked with a context that
// is already done must return the context error without performing any database
// work. This is the mechanism that lets abandoned requests (e.g. a client past
// its deadline) drop out instead of running pointless work under the lock.

// Test_CancelledContext_Add_NoMutation verifies that Add with an already-cancelled
// context returns the context error and does NOT insert a row.
func Test_CancelledContext_Add_NoMutation(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	require.NoError(t, err)
	require.NotNil(t, store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := prefix.New("abc", "192.168.0.0/24", nil)

	// Add under a cancelled context must fail with the context error.
	err = store.Add(ctx, p)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// And it must not have inserted anything: a Get with a live context returns
	// nil (not found), proving the cancelled Add performed no DB mutation.
	got, err := store.Get(context.Background(), "abc", nil)
	require.NoError(t, err)
	assert.Nil(t, got, "cancelled Add must not have inserted a row")
}

// Test_CancelledContext_Get verifies that Get with an already-cancelled context
// returns the context error rather than querying the database.
func Test_CancelledContext_Get(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	require.NoError(t, err)

	// Seed a row with a live context.
	p := prefix.New("abc", "192.168.0.0/24", nil)
	require.NoError(t, store.Add(context.Background(), p))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := store.Get(ctx, "abc", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, got)
}

// Test_CancelledContext_GetChilds verifies that GetChilds with an already-cancelled
// context returns the context error.
func Test_CancelledContext_GetChilds(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	require.NoError(t, err)

	parent := prefix.New("abc", "192.168.0.0/24", nil)
	require.NoError(t, store.Add(context.Background(), parent))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := store.GetChilds(ctx, parent)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, got)
}

// Test_CancelledContext_Delete_NoMutation verifies that Delete with an already-cancelled
// context returns the context error and does NOT remove the row.
func Test_CancelledContext_Delete_NoMutation(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	require.NoError(t, err)

	p := prefix.New("abc", "192.168.0.0/24", nil)
	require.NoError(t, store.Add(context.Background(), p))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = store.Delete(ctx, p)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// The row must still be present.
	got, err := store.Get(context.Background(), "abc", nil)
	require.NoError(t, err)
	assert.NotNil(t, got, "cancelled Delete must not have removed the row")
}

// Test_LiveContext_StillWorks is a control: with a normal (live) context the
// store operates exactly as before, so the context-aware lock does not regress
// the common path.
func Test_LiveContext_StillWorks(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	require.NoError(t, err)

	ctx := context.Background()
	p := prefix.New("abc", "192.168.0.0/24", nil)
	require.NoError(t, store.Add(ctx, p))

	got, err := store.Get(ctx, "abc", nil)
	require.NoError(t, err)
	assert.Equal(t, p, got)

	require.NoError(t, store.Delete(ctx, p))
	got, err = store.Get(ctx, "abc", nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}
