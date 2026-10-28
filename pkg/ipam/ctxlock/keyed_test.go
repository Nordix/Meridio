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
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/ipam/ctxlock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_KeyedMutex_SameKey_MutualExclusion verifies that a second Lock on the
// same key does not acquire while the key is held, and acquires once released.
func Test_KeyedMutex_SameKey_MutualExclusion(t *testing.T) {
	k := ctxlock.NewKeyedMutex()
	require.NoError(t, k.Lock(context.Background(), "a"))

	acquired := make(chan struct{})
	go func() {
		_ = k.Lock(context.Background(), "a")
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second Lock on the same key acquired while the key was held")
	case <-time.After(100 * time.Millisecond):
		// Expected: has not acquired while held.
	}

	k.Unlock("a")

	select {
	case <-acquired:
	case <-time.After(acquireTimeout):
		t.Fatal("second Lock on the same key did not acquire after release")
	}
	k.Unlock("a")
}

// Test_KeyedMutex_DifferentKeys_Independent verifies that holding one key does
// not block acquiring another key.
func Test_KeyedMutex_DifferentKeys_Independent(t *testing.T) {
	k := ctxlock.NewKeyedMutex()
	require.NoError(t, k.Lock(context.Background(), "a"))
	defer k.Unlock("a")

	ctx, cancel := context.WithTimeout(context.Background(), acquireTimeout)
	defer cancel()
	require.NoError(t, k.Lock(ctx, "b"), "a different key must be acquirable while another key is held")
	k.Unlock("b")
}

// Test_KeyedMutex_DeadlineExceeded verifies that waiting for a held key honors
// ctx and returns its error.
func Test_KeyedMutex_DeadlineExceeded(t *testing.T) {
	k := ctxlock.NewKeyedMutex()
	require.NoError(t, k.Lock(context.Background(), "a"))
	defer k.Unlock("a")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := k.Lock(ctx, "a")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// Test_KeyedMutex_Unlock_NotHeld_Panics verifies that releasing an unheld key
// panics, guarding against unbalanced Unlock calls.
func Test_KeyedMutex_Unlock_NotHeld_Panics(t *testing.T) {
	k := ctxlock.NewKeyedMutex()
	assert.Panics(t, func() { k.Unlock("a") })
}
