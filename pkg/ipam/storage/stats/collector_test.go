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
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/ipam/ctxlock"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectorTestTimeout bounds the wait for an expected log entry.
const collectorTestTimeout = 2 * time.Second

// longInterval keeps periodic reports out of tests that do not expect them.
const longInterval = time.Hour

type logEntry struct {
	msg string
	kv  map[string]any
}

// captureSink is a logr.LogSink that sends each Info entry to entries.
type captureSink struct {
	entries chan logEntry
}

func (s *captureSink) Init(logr.RuntimeInfo)          {}
func (s *captureSink) Enabled(int) bool               { return true }
func (s *captureSink) Error(error, string, ...any)    {}
func (s *captureSink) WithValues(...any) logr.LogSink { return s }
func (s *captureSink) WithName(string) logr.LogSink   { return s }
func (s *captureSink) Info(_ int, msg string, kvs ...any) {
	kv := map[string]any{}
	for i := 0; i+1 < len(kvs); i += 2 {
		kv[kvs[i].(string)] = kvs[i+1]
	}
	s.entries <- logEntry{msg: msg, kv: kv}
}

func newTestCollector(t *testing.T, interval time.Duration) (*Collector, chan logEntry) {
	t.Helper()
	entries := make(chan logEntry, 100)
	c, err := NewCollector(logr.New(&captureSink{entries: entries}), interval)
	require.NoError(t, err)
	return c, entries
}

// expectLog waits for the next log entry and checks its message.
func expectLog(t *testing.T, entries <-chan logEntry, msg string) logEntry {
	t.Helper()
	select {
	case e := <-entries:
		require.Equal(t, msg, e.msg)
		return e
	case <-time.After(collectorTestTimeout):
		t.Fatalf("no log entry %q", msg)
		return logEntry{}
	}
}

// expectReport waits for the next log entry, checks that it is a report with
// trigger and returns the report.
func expectReport(t *testing.T, entries <-chan logEntry, trigger string) report {
	t.Helper()
	e := expectLog(t, entries, "Storage statistics")
	r, ok := e.kv["stats"].(report)
	require.True(t, ok, "stats value is a report")
	require.Equal(t, trigger, r.Trigger)
	return r
}

// runCollector starts c.Run and returns its toggle channel, a cancel function
// and a channel closed when Run returns.
func runCollector(c *Collector, enabled bool) (chan os.Signal, context.CancelFunc, <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	toggle := make(chan os.Signal, 1)
	done := make(chan struct{})
	go func() {
		c.Run(ctx, enabled, toggle)
		close(done)
	}()
	return toggle, cancel, done
}

// useStorage does one read and one write through the decorated store and one
// acquisition of the lock.
func useStorage(t *testing.T, c *Collector) {
	t.Helper()
	ctx := context.Background()
	store := NewStore(&fakeStore{}, c.StoreSwitch())
	require.NoError(t, store.Add(ctx, prefix.New("p", "10.0.0.0/24", nil)))
	_, err := store.Get(ctx, "p", nil)
	require.NoError(t, err)
	m := ctxlock.NewWithStats(c.LockSwitch())
	require.NoError(t, m.Lock(ctx))
	m.Unlock()
}

func Test_NewCollector_InvalidInterval(t *testing.T) {
	_, err := NewCollector(logr.Discard(), 0)
	require.Error(t, err)
}

// Test_Collector_Toggle verifies that toggling enables statistics, that
// disabling logs a final report, and that re-enabling starts from zero.
func Test_Collector_Toggle(t *testing.T) {
	c, entries := newTestCollector(t, longInterval)
	toggle, cancel, done := runCollector(c, false)
	defer func() { cancel(); <-done }()

	useStorage(t, c) // disabled, not recorded
	toggle <- syscall.SIGWINCH
	e := expectLog(t, entries, "Storage statistics enabled")
	assert.Equal(t, []string{"<1ms", "<10ms", "<100ms", "<1s", ">=1s"}, e.kv["latencyBuckets"])

	useStorage(t, c)
	toggle <- syscall.SIGWINCH
	r := expectReport(t, entries, triggerDisable)
	assert.Equal(t, int64(1), r.Total.Read.Count)
	assert.Equal(t, int64(1), r.Total.Write.Count)
	assert.Equal(t, int64(1), r.Total.Lock.Acquired)
	expectLog(t, entries, "Storage statistics disabled")

	useStorage(t, c) // disabled, not recorded
	toggle <- syscall.SIGWINCH
	expectLog(t, entries, "Storage statistics enabled")
	toggle <- syscall.SIGWINCH
	r = expectReport(t, entries, triggerDisable)
	assert.Equal(t, int64(0), r.Total.Read.Count)
	assert.Equal(t, int64(0), r.Total.Write.Count)
	assert.Equal(t, int64(0), r.Total.Lock.Acquired)
	expectLog(t, entries, "Storage statistics disabled")
}

// Test_Collector_Shutdown verifies that Run enables statistics at start when
// asked and logs a final report when ctx is done.
func Test_Collector_Shutdown(t *testing.T) {
	c, entries := newTestCollector(t, longInterval)
	_, cancel, done := runCollector(c, true)
	expectLog(t, entries, "Storage statistics enabled")

	useStorage(t, c)
	cancel()
	r := expectReport(t, entries, triggerShutdown)
	assert.Equal(t, int64(1), r.Total.Write.Count)
	select {
	case <-done:
	case <-time.After(collectorTestTimeout):
		t.Fatal("Run did not return")
	}
}

// Test_Collector_Interval verifies that reports are logged periodically while
// enabled.
func Test_Collector_Interval(t *testing.T) {
	c, entries := newTestCollector(t, 10*time.Millisecond)
	_, cancel, done := runCollector(c, true)
	defer func() { cancel(); <-done }()

	expectLog(t, entries, "Storage statistics enabled")
	expectReport(t, entries, triggerInterval)
	expectReport(t, entries, triggerInterval)
}

// Test_Period_Report verifies totals, deltas and peaks across reports.
func Test_Period_Report(t *testing.T) {
	c, _ := newTestCollector(t, longInterval)
	p := c.enable()
	useStorage(t, c)
	r1 := p.report(triggerInterval)

	useStorage(t, c)
	useStorage(t, c)
	r2 := p.report(triggerInterval)

	assert.Equal(t, int64(1), r1.Total.Write.Count)
	assert.Equal(t, int64(1), r1.Delta.Write.Count)
	assert.Equal(t, int64(3), r2.Total.Write.Count)
	assert.Equal(t, int64(2), r2.Delta.Write.Count)
	assert.Equal(t, int64(3), r2.Total.Read.OK)
	assert.Equal(t, int64(2), r2.Delta.Read.OK)
	assert.Equal(t, int64(3), r2.Total.Lock.Acquired)
	assert.Equal(t, int64(2), r2.Delta.Lock.Acquired)
	assert.Equal(t, int64(3), r2.Total.Read.Buckets[0], "fake store calls are below 1ms")
	assert.Equal(t, int64(2), r2.Delta.Read.Buckets[0])

	r3 := p.report(triggerInterval)
	assert.Equal(t, opCounts{Time: "0s"}, r3.Delta.Write, "nothing since previous report")
	assert.Equal(t, "0s", r3.Peak.Write, "peak reset by previous report")
}
