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

package ctxlock

import (
	"context"
	"sync"
)

// KeyedMutex is a set of Mutexes indexed by key. Holders of the same key are
// serialized, while different keys proceed independently. Lock has the same
// context-aware, FIFO-fair semantics as Mutex.Lock.
//
// The Mutex for a key is created on first use and kept for the lifetime of the
// KeyedMutex, so KeyedMutex is intended for a bounded key space.
//
// Implementation note: the key-to-Mutex map is a plain map guarded by a
// sync.Mutex rather than a sync.Map. sync.Map would also fit (keys are written
// once and then only read) and would make lookups of existing keys lock-free,
// using a Load fast path and LoadOrStore only for a new key. That gain is not
// measurable here: the map lock is held only for a lookup, which is negligible
// next to the work done while holding a key. A plain map is preferred because
// it keeps the code typed, whereas sync.Map stores untyped values that need a
// type assertion on every lookup.
type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]*Mutex
}

// NewKeyedMutex returns a ready-to-use KeyedMutex with no keys held.
func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{locks: map[string]*Mutex{}}
}

// Lock acquires the Mutex for key, blocking until it is available or ctx is
// done. On success the caller must call Unlock(key) exactly once. If ctx is
// done first, nothing is acquired and ctx.Err() is returned.
func (k *KeyedMutex) Lock(ctx context.Context, key string) error {
	return k.get(key).Lock(ctx)
}

// Unlock releases the Mutex for key. It must only be called by a goroutine that
// holds key (i.e. after a Lock(ctx, key) call that returned nil).
func (k *KeyedMutex) Unlock(key string) {
	k.get(key).Unlock()
}

// get returns the Mutex for key, creating it on first use.
func (k *KeyedMutex) get(key string) *Mutex {
	k.mu.Lock()
	defer k.mu.Unlock()
	m, ok := k.locks[key]
	if !ok {
		m = New()
		k.locks[key] = m
	}
	return m
}
