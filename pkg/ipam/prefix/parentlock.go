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

// parentLocks serializes allocations per parent prefix, keyed by the parent's
// CIDR. IPAM uses a single store per process, so a process-wide registry has the
// same scope as the store. Only trench, conduit and node prefixes act as
// parents, so the key space is bounded by the prefix plan and entries are never
// evicted.
var parentLocks = ctxlock.NewKeyedMutex()

// WithParentLock decorates allocate so that allocations under the same parent
// are serialized within this process. This makes the read children, add, verify
// sequence of an allocation atomic with respect to other allocations under the
// same parent, so the first free candidate normally succeeds instead of
// colliding and retrying. Allocations under different parents are not affected.
// Waiting for the lock honors ctx cancellation.
//
// The undecorated allocator remains correct on its own (the store's uniqueness
// constraints and the retry loop resolve races); the lock only avoids wasted
// retry rounds between same-parent allocations.
func WithParentLock(allocate AllocateFunc) AllocateFunc {
	return func(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
		key := parent.GetCidr()
		if err := parentLocks.Lock(ctx, key); err != nil {
			return nil, fmt.Errorf("failed to lock parent while allocating prefix (name: %s ; length: %d ; parent: %s): %w", name, length, parent.GetName(), err)
		}
		defer parentLocks.Unlock(key)
		return allocate(ctx, parent, name, length, store, blocklist)
	}
}

var allocateWithParentLock = WithParentLock(AllocateWithBlocklist)

// AllocateWithParentLock is AllocateWithBlocklist with allocations under the
// same parent serialized (see WithParentLock).
func AllocateWithParentLock(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
	return allocateWithParentLock(ctx, parent, name, length, store, blocklist)
}
