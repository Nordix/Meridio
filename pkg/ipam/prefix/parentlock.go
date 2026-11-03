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

package prefix

import (
	"context"
	"fmt"

	"github.com/nordix/meridio/pkg/ipam/ctxlock"
	"github.com/nordix/meridio/pkg/ipam/types"
)

// AllocateFunc has the signature of AllocateWithBlocklist.
type AllocateFunc func(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error)

// ParentLocker is implemented by stores that serialize allocations per parent
// prefix (see WithParentLock and ParentLockingStore).
type ParentLocker interface {
	// LockParent blocks until no other allocation holds parent, or ctx is
	// done. On success the caller must call unlock exactly once.
	LockParent(ctx context.Context, parent types.Prefix) (unlock func(), err error)
}

// ParentLockingStore wraps a store and adds per-parent allocation locks to it
// (ParentLocker). Storage operations are passed through unchanged. The locks
// belong to the wrapped store, so their scope is exactly that store.
//
// It must be the outermost wrapper of the store given to the allocator:
// WithParentLock finds the locks by asserting the store it receives, so a
// decorator wrapped around it would hide them and turn locking off.
type ParentLockingStore struct {
	types.Storage
	// locks is keyed by the parent's CIDR rather than its name-based ID. The
	// CIDR identifies the subnet itself and stays the same for the trench's
	// lifetime, whereas the owner, and so the ID, can change (e.g. when a node
	// prefix is garbage collected and its subnet is later reallocated).
	// Keys are never evicted: only trench, conduit and node prefixes act as
	// parents, so the key space is bounded by the prefix plan.
	locks *ctxlock.KeyedMutex
}

// NewParentLockingStore returns store with per-parent allocation locks.
func NewParentLockingStore(store types.Storage) *ParentLockingStore {
	return &ParentLockingStore{Storage: store, locks: ctxlock.NewKeyedMutex()}
}

// LockParent implements ParentLocker.
func (s *ParentLockingStore) LockParent(ctx context.Context, parent types.Prefix) (func(), error) {
	key := parent.GetCidr()
	if err := s.locks.Lock(ctx, key); err != nil {
		return nil, err
	}
	return func() { s.locks.Unlock(key) }, nil
}

// WithParentLock decorates allocate so that allocations under the same parent
// are serialized, provided the store implements ParentLocker; otherwise
// allocate runs unlocked. This makes the read children, add, verify sequence
// of an allocation atomic with respect to other allocations under the same
// parent, so the first free candidate normally succeeds instead of colliding
// and retrying. Allocations under different parents are not affected. Waiting
// for the lock honors ctx cancellation.
//
// The undecorated allocator remains correct on its own (the store's uniqueness
// constraints and the retry loop resolve races); the lock only avoids wasted
// retry rounds between same-parent allocations.
//
// Lock order: the parent lock is taken before, and held across, the store
// operations of an allocation. A store operation must never allocate, or the
// two locks could deadlock.
func WithParentLock(allocate AllocateFunc) AllocateFunc {
	return func(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
		if locker, ok := store.(ParentLocker); ok {
			unlock, err := locker.LockParent(ctx, parent)
			if err != nil {
				return nil, fmt.Errorf("failed to lock parent while allocating prefix (name: %s ; length: %d ; parent: %s): %w", name, length, parent.GetName(), err)
			}
			defer unlock()
		}
		return allocate(ctx, parent, name, length, store, blocklist)
	}
}

var allocateWithParentLock = WithParentLock(AllocateWithBlocklist)

// AllocateWithParentLock is AllocateWithBlocklist with allocations under the
// same parent serialized when the store implements ParentLocker (see
// WithParentLock).
func AllocateWithParentLock(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
	return allocateWithParentLock(ctx, parent, name, length, store, blocklist)
}
