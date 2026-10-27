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

// Package ctxlock provides a mutual-exclusion lock whose acquisition honors
// context cancellation and grants the lock to waiters in first-in-first-out
// order.
//
// Unlike sync.Mutex, a caller waiting for a Mutex can abandon the wait when its
// context is done (for example, because the client already hit its deadline). A
// request that has been given up on therefore stops queuing behind the lock
// instead of waiting its full turn and then performing now-pointless work. This
// lets a contended lock's wait queue drain under load rather than filling with
// abandoned waiters.
//
// Acquisitions are FIFO-fair: a newly arriving caller cannot barge ahead of
// goroutines already waiting. This bounds the spread of acquisition latencies
// under heavy contention (no waiter is repeatedly overtaken), unlike the
// barging behaviour of a plain buffered-channel lock, but similar to sync.Mutex
// in starvation mode.
//
// Cancellation applies only to the wait for the lock, not to a holder already
// inside its critical section. Once Lock returns nil, the holder runs to
// completion and must call Unlock.
//
// Mutex is implemented on golang.org/x/sync/semaphore (a weighted semaphore of
// size 1), which provides the FIFO waiter list and the context-aware acquire.
package ctxlock

import (
	"context"

	"golang.org/x/sync/semaphore"
)

// Mutex is a mutual-exclusion lock with a context-aware, FIFO-fair Lock. The
// zero value is not usable; call New to construct one.
//
// A Mutex is a plain one-holder-at-a-time lock. It is not reentrant.
type Mutex struct {
	sem *semaphore.Weighted
}

// New returns a ready-to-use, unlocked Mutex.
func New() *Mutex {
	return &Mutex{sem: semaphore.NewWeighted(1)}
}

// Lock acquires the mutex, blocking until it is available or ctx is done.
//
// On success it returns nil and the caller holds the lock and must call Unlock
// exactly once. If ctx is done before the lock is acquired, Lock acquires
// nothing and returns ctx.Err() (context.Canceled or context.DeadlineExceeded);
// the caller must not call Unlock in that case.
//
// As a fast path, Lock returns ctx.Err() if ctx is already done, without
// acquiring, so an already-abandoned request performs no work. While waiting,
// Lock preserves FIFO order, and a cancelled waiter removes itself from the
// queue so it stops blocking those behind it.
func (m *Mutex) Lock(ctx context.Context) error {
	return m.sem.Acquire(ctx, 1)
}

// Unlock releases the mutex. It must be called only by a goroutine that
// currently holds the lock (i.e. after a Lock call that returned nil).
// Releasing a lock that is not held panics.
func (m *Mutex) Unlock() {
	m.sem.Release(1)
}
