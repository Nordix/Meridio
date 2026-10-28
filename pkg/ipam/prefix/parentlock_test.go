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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const parentLockTestTimeout = 2 * time.Second

// Test_WithParentLock_SameParent_Serialized verifies that decorated
// allocations under the same parent never run concurrently. The fake allocator
// records the maximum number of concurrent invocations; the short sleep inside
// it only widens the window in which an overlap would be observed.
func Test_WithParentLock_SameParent_Serialized(t *testing.T) {
	parent := prefix.New("parent", "10.10.0.0/16", nil)

	var inFlight, maxInFlight atomic.Int32
	allocate := prefix.WithParentLock(func(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
		n := inFlight.Add(1)
		for {
			m := maxInFlight.Load()
			if n <= m || maxInFlight.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		return prefix.New(name, "10.10.0.0/24", parent), nil
	})

	const workers = 10
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, err := allocate(context.Background(), parent, "child", 24, nil, nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), maxInFlight.Load(), "same-parent allocations must be serialized")
}

// Test_WithParentLock_DifferentParents_Concurrent verifies that allocations
// under different parents are not serialized. Each allocation signals that it
// is running (holding its parent's lock) and then waits for the other one to be
// running too. Both can only succeed if the two run at the same time; if the
// parents were serialized, whichever ran first would wait in vain and time out.
func Test_WithParentLock_DifferentParents_Concurrent(t *testing.T) {
	parentA := prefix.New("parent-a", "10.11.0.0/16", nil)
	parentB := prefix.New("parent-b", "10.12.0.0/16", nil)

	aEntered := make(chan struct{})
	bEntered := make(chan struct{})

	// rendezvous signals entry via self, then waits until other has entered.
	rendezvous := func(ctx context.Context, self, other chan struct{}) error {
		close(self)
		select {
		case <-other:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	allocate := prefix.WithParentLock(func(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
		self, other := aEntered, bEntered
		if parent.Equals(parentB) {
			self, other = bEntered, aEntered
		}
		if err := rendezvous(ctx, self, other); err != nil {
			return nil, err
		}
		return prefix.New(name, parent.GetCidr(), parent), nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), parentLockTestTimeout)
	defer cancel()

	errA := make(chan error, 1)
	go func() {
		_, err := allocate(ctx, parentA, "child", 24, nil, nil)
		errA <- err
	}()
	_, errB := allocate(ctx, parentB, "child", 24, nil, nil)

	require.NoError(t, errB, "allocation under parent B must run concurrently with parent A")
	require.NoError(t, <-errA, "allocation under parent A must run concurrently with parent B")
}

// Test_WithParentLock_CancelWhileWaiting verifies that waiting for a parent
// held by another allocation honors ctx: the waiting call returns the ctx error
// and its allocation is never executed.
func Test_WithParentLock_CancelWhileWaiting(t *testing.T) {
	parent := prefix.New("parent", "10.13.0.0/16", nil)

	holding := make(chan struct{}) // closed once the first allocation holds the lock
	release := make(chan struct{})
	var calls atomic.Int32
	allocate := prefix.WithParentLock(func(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
		if calls.Add(1) == 1 {
			close(holding)
			<-release
		}
		return prefix.New(name, "10.13.0.0/24", parent), nil
	})

	firstDone := make(chan error, 1)
	go func() {
		_, err := allocate(context.Background(), parent, "first", 24, nil, nil)
		firstDone <- err
	}()

	select {
	case <-holding:
	case <-time.After(parentLockTestTimeout):
		t.Fatal("first allocation did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := allocate(ctx, parent, "second", 24, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), calls.Load(), "a cancelled waiter must not run its allocation")

	close(release)
	require.NoError(t, <-firstDone)
}

// countingStore counts Add and Delete calls on the wrapped store.
type countingStore struct {
	types.Storage
	adds    atomic.Int32
	deletes atomic.Int32
}

func (s *countingStore) Add(ctx context.Context, p types.Prefix) error {
	s.adds.Add(1)
	return s.Storage.Add(ctx, p)
}

func (s *countingStore) Delete(ctx context.Context, p types.Prefix) error {
	s.deletes.Add(1)
	return s.Storage.Delete(ctx, p)
}

// Test_AllocateWithParentLock_ConcurrentSameParent_MemoryStore shows the
// effect on the allocator itself. Without the lock, concurrent same-parent
// attempts against the memory store keep colliding on the same candidate and
// end with no stable allocation (IPv4_Memory_Storage_Collision in
// Test_Prefix_Allocate_Concurrency). With same-parent allocations serialized,
// each attempt sees the previous ones' results, so 16 concurrent contenders
// fill the /24 parent with its 16 /28 subnets, each on its first candidate:
// exactly one Add per contender and no Delete (no collision rollback).
func Test_AllocateWithParentLock_ConcurrentSameParent_MemoryStore(t *testing.T) {
	parent, delayed := setupTestEnv(t, "169.17.0.0/24", setupDelayedMemoryStore)
	store := &countingStore{Storage: delayed}

	const contenders = 16 // number of /28 subnets in a /24
	expected := map[string]struct{}{}
	for i := 0; i < contenders; i++ {
		expected[fmt.Sprintf("169.17.0.%d/28", i*16)] = struct{}{}
	}

	var mu sync.Mutex
	allocated := map[string]string{} // CIDR -> name
	var wg sync.WaitGroup
	wg.Add(contenders)
	for i := 0; i < contenders; i++ {
		go func(name string) {
			defer wg.Done()
			p, err := prefix.AllocateWithParentLock(context.Background(), parent, name, 28, store, nil)
			if !assert.NoError(t, err) {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			assert.NotContains(t, allocated, p.GetCidr(), "duplicate CIDR allocated")
			allocated[p.GetCidr()] = p.GetName()
		}(fmt.Sprintf("child-%d", i))
	}
	wg.Wait()

	assert.Len(t, allocated, contenders)
	for cidr := range expected {
		assert.Contains(t, allocated, cidr)
	}
	assert.Equal(t, int32(contenders), store.adds.Load(), "each contender should succeed on its first candidate")
	assert.Equal(t, int32(0), store.deletes.Load(), "no collision rollback expected")
}
