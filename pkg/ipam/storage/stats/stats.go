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

// Package stats provides optional statistics of the IPAM storage, for
// diagnosing storage latency and lock contention from the logs.
//
// Store decorates a types.Storage and counts reads and writes by outcome and
// latency. The storage lock records its acquisitions through a
// ctxlock.StatsSwitch. Both are controlled by switches that are disabled by
// default; while disabled, the cost per call is one atomic load.
//
// Collector owns the switches. It turns statistics on and off on request and,
// while they are on, logs a report periodically, on disable and on shutdown.
// Each enable starts from zero.
package stats

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/nordix/meridio/pkg/ipam/prefix"
)

// latencyBounds are the upper bounds of the latency buckets. An operation
// lands in the first bucket whose bound it is below, or in the last bucket if
// it takes latencyBounds[len(latencyBounds)-1] or longer.
var latencyBounds = [...]time.Duration{
	time.Millisecond,
	10 * time.Millisecond,
	100 * time.Millisecond,
	time.Second,
}

// Buckets holds one counter per latency bucket (see latencyBounds).
type Buckets [len(latencyBounds) + 1]int64

// Switch turns storage statistics on and off. Each Enable installs a new
// StoreStats, so collection always starts from zero. An operation records into
// the StoreStats installed when it started. The zero value and a nil Switch
// are disabled.
type Switch struct {
	current atomic.Pointer[StoreStats]
}

// Enable installs and returns a new StoreStats.
func (w *Switch) Enable() *StoreStats {
	s := &StoreStats{}
	w.current.Store(s)
	return s
}

// Disable stops collection.
func (w *Switch) Disable() {
	w.current.Store(nil)
}

// Stats returns the installed StoreStats, or nil if collection is disabled or
// w is nil.
func (w *Switch) Stats() *StoreStats {
	if w == nil {
		return nil
	}
	return w.current.Load()
}

// StoreStats holds the statistics of one enabled period of a Switch. Get and
// GetChilds count as reads, Add, Update and Delete as writes.
type StoreStats struct {
	Read  OpStats
	Write OpStats
}

// OpStats counts operations of one kind by outcome and latency. It is safe for
// concurrent use.
type OpStats struct {
	ok           atomic.Int64
	ctxErr       atomic.Int64
	cidrConflict atomic.Int64
	idConflict   atomic.Int64
	otherErr     atomic.Int64
	totalNanos   atomic.Int64
	peakNanos    atomic.Int64
	buckets      [len(latencyBounds) + 1]atomic.Int64
}

// OpSnapshot is a point-in-time view of an OpStats. The fields are read one by
// one, so a snapshot taken under load is approximate.
type OpSnapshot struct {
	// Count is the number of operations, the sum of the outcome counters.
	Count int64
	// OK counts operations that returned no error.
	OK int64
	// CtxErr counts operations that failed because their context was done.
	CtxErr int64
	// CIDRConflict counts Add calls rejected for an overlapping CIDR.
	CIDRConflict int64
	// IDConflict counts Add calls rejected for an existing primary ID.
	IDConflict int64
	// OtherErr counts operations that failed for any other reason.
	OtherErr int64
	// TotalTime is the time spent in the operations.
	TotalTime time.Duration
	// Buckets counts the operations per latency bucket.
	Buckets Buckets
	// PeakTime is the longest operation since the previous snapshot.
	PeakTime time.Duration
}

// Snapshot returns the current statistics and starts a new peak period.
func (o *OpStats) Snapshot() OpSnapshot {
	s := OpSnapshot{
		OK:           o.ok.Load(),
		CtxErr:       o.ctxErr.Load(),
		CIDRConflict: o.cidrConflict.Load(),
		IDConflict:   o.idConflict.Load(),
		OtherErr:     o.otherErr.Load(),
		TotalTime:    time.Duration(o.totalNanos.Load()),
		PeakTime:     time.Duration(o.peakNanos.Swap(0)),
	}
	s.Count = s.OK + s.CtxErr + s.CIDRConflict + s.IDConflict + s.OtherErr
	for i := range o.buckets {
		s.Buckets[i] = o.buckets[i].Load()
	}
	return s
}

// observe records an operation that took d and returned err.
func (o *OpStats) observe(d time.Duration, err error) {
	switch {
	case err == nil:
		o.ok.Add(1)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		o.ctxErr.Add(1)
	case errors.Is(err, prefix.ErrCIDRConflict):
		o.cidrConflict.Add(1)
	case errors.Is(err, prefix.ErrPrimaryIDConflict):
		o.idConflict.Add(1)
	default:
		o.otherErr.Add(1)
	}
	o.totalNanos.Add(int64(d))
	o.buckets[bucketIndex(d)].Add(1)
	storeMax(&o.peakNanos, int64(d))
}

// bucketIndex returns the latency bucket of d.
func bucketIndex(d time.Duration) int {
	for i, bound := range latencyBounds {
		if d < bound {
			return i
		}
	}
	return len(latencyBounds)
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
