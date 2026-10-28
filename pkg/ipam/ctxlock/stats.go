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
	"sync/atomic"
	"time"
)

// StatsSwitch turns statistics collection on and off for the Mutexes it is
// attached to. Each Enable installs a new LockStats, so collection always
// starts from zero. A Lock call records into the LockStats that was installed
// when it started, so switching does not disturb calls in progress. The zero
// value is disabled and ready to use.
type StatsSwitch struct {
	current atomic.Pointer[LockStats]
}

// Enable installs and returns a new LockStats.
func (w *StatsSwitch) Enable() *LockStats {
	s := &LockStats{}
	w.current.Store(s)
	return s
}

// Disable stops collection.
func (w *StatsSwitch) Disable() {
	w.current.Store(nil)
}

// Stats returns the installed LockStats, or nil if collection is disabled.
func (w *StatsSwitch) Stats() *LockStats {
	return w.current.Load()
}

// LockStats holds the acquisition statistics of one enabled period of a
// StatsSwitch. It is safe for concurrent use.
type LockStats struct {
	acquired      atomic.Int64
	contended     atomic.Int64
	abandoned     atomic.Int64
	waitNanos     atomic.Int64
	waiters       atomic.Int64
	peakWaiters   atomic.Int64
	peakWaitNanos atomic.Int64
}

// LockSnapshot is a point-in-time view of a LockStats. The fields are read
// one by one, so a snapshot taken under load is approximate.
type LockSnapshot struct {
	// Acquired counts Lock calls that acquired the lock.
	Acquired int64
	// Contended counts acquisitions that had to wait for the lock.
	Contended int64
	// Abandoned counts Lock calls that returned a context error, whether the
	// context was done on entry or while waiting.
	Abandoned int64
	// TotalWait is the time spent waiting, including abandoned waits.
	TotalWait time.Duration
	// Waiters is the number of callers waiting at the time of the snapshot.
	Waiters int64
	// PeakWaiters is the highest number of concurrent waiters since the
	// previous snapshot.
	PeakWaiters int64
	// PeakWait is the longest single wait since the previous snapshot.
	PeakWait time.Duration
}

// Snapshot returns the current statistics and starts a new peak period.
func (s *LockStats) Snapshot() LockSnapshot {
	waiters := s.waiters.Load()
	return LockSnapshot{
		Acquired:    s.acquired.Load(),
		Contended:   s.contended.Load(),
		Abandoned:   s.abandoned.Load(),
		TotalWait:   time.Duration(s.waitNanos.Load()),
		Waiters:     waiters,
		PeakWaiters: s.peakWaiters.Swap(waiters),
		PeakWait:    time.Duration(s.peakWaitNanos.Swap(0)),
	}
}

// startWait records a caller that starts waiting for the lock.
func (s *LockStats) startWait() time.Time {
	storeMax(&s.peakWaiters, s.waiters.Add(1))
	return time.Now()
}

// endWait records the outcome of a wait that started at start.
func (s *LockStats) endWait(start time.Time, err error) {
	d := int64(time.Since(start))
	s.waiters.Add(-1)
	s.waitNanos.Add(d)
	storeMax(&s.peakWaitNanos, d)
	if err != nil {
		s.abandoned.Add(1)
		return
	}
	s.acquired.Add(1)
	s.contended.Add(1)
}

// storeMax sets v to n if n is larger.
func storeMax(v *atomic.Int64, n int64) {
	for {
		cur := v.Load()
		if n <= cur || v.CompareAndSwap(cur, n) {
			return
		}
	}
}
