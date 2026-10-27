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

package prefix_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// releaseRaceStore makes the first Add fail with a primary-ID conflict, while
// simulating that in the meantime the conflicting same-name row was released
// and another allocation took the candidate CIDR.
type releaseRaceStore struct {
	types.Storage
	parent  types.Prefix
	raced   bool
	adds    int
	deletes int
}

func (s *releaseRaceStore) Add(ctx context.Context, p types.Prefix) error {
	s.adds++
	if !s.raced {
		s.raced = true
		if err := s.Storage.Add(ctx, prefix.New("other", p.GetCidr(), s.parent)); err != nil {
			return err
		}
		return prefix.ErrPrimaryIDConflict
	}
	return s.Storage.Add(ctx, p)
}

func (s *releaseRaceStore) Delete(ctx context.Context, p types.Prefix) error {
	s.deletes++
	return s.Storage.Delete(ctx, p)
}

// Test_Allocate_SameName_ReleasedMeanwhile_RefreshesChildren verifies that when
// a primary-ID conflict is followed by the same-name row being gone (released
// concurrently), the retry works on refreshed children: it skips the candidate
// taken in the meantime and succeeds on the next one, without a colliding Add
// and its rollback.
func Test_Allocate_SameName_ReleasedMeanwhile_RefreshesChildren(t *testing.T) {
	parent := prefix.New("parent", "169.18.0.0/24", nil)
	store := &releaseRaceStore{Storage: memory.New(), parent: parent}

	p, err := prefix.Allocate(context.TODO(), parent, "child", 25, store)
	require.NoError(t, err)
	assert.Equal(t, "169.18.0.128/25", p.GetCidr())
	assert.Equal(t, 2, store.adds, "the retry should succeed on its first candidate")
	assert.Equal(t, 0, store.deletes, "no collision rollback expected")
}

// Test_Allocate_SameName_Idempotent verifies that, when a prefix with the
// requested name already exists under the parent (as happens when a concurrent
// or earlier attempt inserted it first), Allocate returns the stored prefix
// instead of failing with a primary-ID uniqueness error. This exercises the
// sqlite store's ErrPrimaryIDConflict path and the allocator's idempotent
// handling of it.
func Test_Allocate_SameName_Idempotent(t *testing.T) {
	store := setupSqliteStore(t)
	parent := prefix.New("parent", "169.16.0.0/16", nil)

	// First allocation establishes the prefix for "child-a".
	first, err := prefix.Allocate(context.TODO(), parent, "child-a", 24, store)
	require.NoError(t, err)
	require.NotNil(t, first)

	// A second allocation for the same name must not fail on the primary-ID
	// conflict; it must return the already-stored prefix (same CIDR, same name).
	second, err := prefix.Allocate(context.TODO(), parent, "child-a", 24, store)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.GetName(), second.GetName())
	assert.Equal(t, first.GetCidr(), second.GetCidr())
}

// Test_Allocate_SameName_Idempotent_DifferentLength verifies the idempotent
// path returns the stored prefix even when the second request asks for a
// different length: the stored entry is authoritative.
func Test_Allocate_SameName_Idempotent_DifferentLength(t *testing.T) {
	store := setupSqliteStore(t)
	parent := prefix.New("parent", "169.16.0.0/16", nil)

	first, err := prefix.Allocate(context.TODO(), parent, "child-a", 24, store)
	require.NoError(t, err)
	require.NotNil(t, first)

	// Same name, different requested length: still returns the stored prefix.
	second, err := prefix.Allocate(context.TODO(), parent, "child-a", 28, store)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, first.GetCidr(), second.GetCidr(),
		"idempotent allocation must return the stored CIDR regardless of requested length")
}
