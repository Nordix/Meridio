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

// Test_Add_SameName_PrimaryIDConflict verifies that inserting a second prefix
// with the same name and parent (and therefore the same primary ID) but a
// different CIDR returns ErrPrimaryIDConflict, not ErrCIDRConflict and not a raw
// error. This is the condition the allocator relies on to resolve same-name
// races idempotently. (The CIDR-conflict branch is covered by
// Test_CIDR_Uniqueness_Constraint.)
func Test_Add_SameName_PrimaryIDConflict(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	require.NoError(t, err)

	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	require.NoError(t, store.Add(context.Background(), p1))

	// Same name+parent, different CIDR -> primary-ID (not CIDR) uniqueness.
	p2 := prefix.New("abc", "192.168.1.0/24", nil)
	err = store.Add(context.Background(), p2)
	require.Error(t, err)
	assert.ErrorIs(t, err, sqlite.ErrPrimaryIDConflict)
	assert.NotErrorIs(t, err, sqlite.ErrCIDRConflict)

	// The original row is unchanged.
	got, err := store.Get(context.Background(), "abc", nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "192.168.0.0/24", got.GetCidr())
}
