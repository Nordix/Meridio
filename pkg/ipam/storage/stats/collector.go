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

package stats

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/ipam/ctxlock"
)

// Report triggers.
const (
	triggerInterval = "interval"
	triggerDisable  = "disable"
	triggerShutdown = "shutdown"
)

// Collector turns the statistics of the storage and of its lock on and off,
// and logs them periodically while they are enabled.
type Collector struct {
	logger   logr.Logger
	interval time.Duration
	lock     ctxlock.StatsSwitch
	store    Switch
}

// NewCollector returns a Collector that logs with logger every interval while
// statistics are enabled. Statistics are disabled until Run enables them.
func NewCollector(logger logr.Logger, interval time.Duration) (*Collector, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("storage statistics interval must be positive: %s", interval)
	}
	return &Collector{
		logger:   logger.WithValues("class", "StorageStats"),
		interval: interval,
	}, nil
}

// LockSwitch returns the switch for the storage lock, or nil if c is nil.
func (c *Collector) LockSwitch() *ctxlock.StatsSwitch {
	if c == nil {
		return nil
	}
	return &c.lock
}

// StoreSwitch returns the switch for the storage operations, or nil if c is
// nil.
func (c *Collector) StoreSwitch() *Switch {
	if c == nil {
		return nil
	}
	return &c.store
}

// Run enables statistics at start if enabled is true, and then switches them
// on or off on each value received from toggle. While enabled, it logs a report
// every interval and when statistics are disabled. Run returns when ctx is
// done, after logging a final report if statistics are enabled.
func (c *Collector) Run(ctx context.Context, enabled bool, toggle <-chan os.Signal) {
	var p *period
	var ticker *time.Ticker
	var tick <-chan time.Time
	start := func() {
		p = c.enable()
		ticker = time.NewTicker(c.interval)
		tick = ticker.C
	}
	if enabled {
		start()
	}
	for {
		select {
		case <-ctx.Done():
			if p != nil {
				ticker.Stop()
				c.log(p, triggerShutdown)
			}
			return
		case <-toggle:
			if p == nil {
				start()
				continue
			}
			ticker.Stop()
			tick = nil
			c.log(p, triggerDisable)
			c.disable()
			p = nil
		case <-tick:
			c.log(p, triggerInterval)
		}
	}
}

// period is one enabled period of the statistics.
type period struct {
	lock  *ctxlock.LockStats
	store *StoreStats
	start time.Time
	last  time.Time // time of the previous report
	prev  totals    // totals of the previous report
}

// totals are the cumulative counters of a period.
type totals struct {
	lock  ctxlock.LockSnapshot
	read  OpSnapshot
	write OpSnapshot
}

func (c *Collector) enable() *period {
	now := time.Now()
	p := &period{
		lock:  c.lock.Enable(),
		store: c.store.Enable(),
		start: now,
		last:  now,
	}
	c.logger.Info("Storage statistics enabled", "interval", c.interval.String(), "latencyBuckets", bucketLabels())
	return p
}

func (c *Collector) disable() {
	c.lock.Disable()
	c.store.Disable()
	c.logger.Info("Storage statistics disabled")
}

func (c *Collector) log(p *period, trigger string) {
	c.logger.Info("Storage statistics", "stats", p.report(trigger))
}

// report is the logged statistics. Total covers the period since statistics
// were enabled, Delta and Peak the time since the previous report.
type report struct {
	Trigger     string  `json:"trigger"`
	EnabledFor  string  `json:"enabledFor"`
	SinceLast   string  `json:"sinceLast"`
	LockWaiters int64   `json:"lockWaiters"`
	Total       section `json:"total"`
	Delta       section `json:"delta"`
	Peak        peak    `json:"peak"`
}

type section struct {
	Read  opCounts   `json:"read"`
	Write opCounts   `json:"write"`
	Lock  lockCounts `json:"lock"`
}

type opCounts struct {
	Count        int64   `json:"count"`
	OK           int64   `json:"ok"`
	CtxErr       int64   `json:"ctxErr"`
	CIDRConflict int64   `json:"cidrConflict"`
	IDConflict   int64   `json:"idConflict"`
	OtherErr     int64   `json:"otherErr"`
	Time         string  `json:"time"`
	Buckets      Buckets `json:"buckets"`
}

type lockCounts struct {
	Acquired  int64  `json:"acquired"`
	Contended int64  `json:"contended"`
	Abandoned int64  `json:"abandoned"`
	Wait      string `json:"wait"`
}

type peak struct {
	Read        string `json:"read"`
	Write       string `json:"write"`
	LockWait    string `json:"lockWait"`
	LockWaiters int64  `json:"lockWaiters"`
}

// report takes snapshots, builds the report and starts a new report period.
func (p *period) report(trigger string) report {
	now := time.Now()
	cur := totals{
		lock:  p.lock.Snapshot(),
		read:  p.store.Read.Snapshot(),
		write: p.store.Write.Snapshot(),
	}
	r := report{
		Trigger:     trigger,
		EnabledFor:  formatDuration(now.Sub(p.start)),
		SinceLast:   formatDuration(now.Sub(p.last)),
		LockWaiters: cur.lock.Waiters,
		Total:       newSection(cur),
		Delta:       newSection(cur.sub(p.prev)),
		Peak: peak{
			Read:        formatDuration(cur.read.PeakTime),
			Write:       formatDuration(cur.write.PeakTime),
			LockWait:    formatDuration(cur.lock.PeakWait),
			LockWaiters: cur.lock.PeakWaiters,
		},
	}
	p.prev = cur
	p.last = now
	return r
}

// sub returns the counters of t minus those of prev.
func (t totals) sub(prev totals) totals {
	return totals{
		lock: ctxlock.LockSnapshot{
			Acquired:  t.lock.Acquired - prev.lock.Acquired,
			Contended: t.lock.Contended - prev.lock.Contended,
			Abandoned: t.lock.Abandoned - prev.lock.Abandoned,
			TotalWait: t.lock.TotalWait - prev.lock.TotalWait,
		},
		read:  t.read.sub(prev.read),
		write: t.write.sub(prev.write),
	}
}

// sub returns the counters of s minus those of prev.
func (s OpSnapshot) sub(prev OpSnapshot) OpSnapshot {
	d := OpSnapshot{
		Count:        s.Count - prev.Count,
		OK:           s.OK - prev.OK,
		CtxErr:       s.CtxErr - prev.CtxErr,
		CIDRConflict: s.CIDRConflict - prev.CIDRConflict,
		IDConflict:   s.IDConflict - prev.IDConflict,
		OtherErr:     s.OtherErr - prev.OtherErr,
		TotalTime:    s.TotalTime - prev.TotalTime,
	}
	for i := range s.Buckets {
		d.Buckets[i] = s.Buckets[i] - prev.Buckets[i]
	}
	return d
}

func newSection(t totals) section {
	return section{
		Read:  newOpCounts(t.read),
		Write: newOpCounts(t.write),
		Lock: lockCounts{
			Acquired:  t.lock.Acquired,
			Contended: t.lock.Contended,
			Abandoned: t.lock.Abandoned,
			Wait:      formatDuration(t.lock.TotalWait),
		},
	}
}

func newOpCounts(s OpSnapshot) opCounts {
	return opCounts{
		Count:        s.Count,
		OK:           s.OK,
		CtxErr:       s.CtxErr,
		CIDRConflict: s.CIDRConflict,
		IDConflict:   s.IDConflict,
		OtherErr:     s.OtherErr,
		Time:         formatDuration(s.TotalTime),
		Buckets:      s.Buckets,
	}
}

// bucketLabels returns a label per latency bucket, e.g. "<1ms" and ">=1s".
func bucketLabels() []string {
	labels := make([]string, 0, len(latencyBounds)+1)
	for _, bound := range latencyBounds {
		labels = append(labels, "<"+bound.String())
	}
	return append(labels, ">="+latencyBounds[len(latencyBounds)-1].String())
}

func formatDuration(d time.Duration) string {
	return d.Round(time.Microsecond).String()
}
