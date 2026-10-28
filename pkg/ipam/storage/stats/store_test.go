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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore returns err from every operation.
type fakeStore struct {
	err error
}

func (f *fakeStore) Add(context.Context, types.Prefix) error    { return f.err }
func (f *fakeStore) Update(context.Context, types.Prefix) error { return f.err }
func (f *fakeStore) Delete(context.Context, types.Prefix) error { return f.err }
func (f *fakeStore) Get(context.Context, string, types.Prefix) (types.Prefix, error) {
	return nil, f.err
}
func (f *fakeStore) GetChilds(context.Context, types.Prefix) ([]types.Prefix, error) {
	return nil, f.err
}

// callAll calls each storage operation once.
func callAll(s types.Storage) {
	ctx := context.Background()
	p := prefix.New("p", "10.0.0.0/24", nil)
	_ = s.Add(ctx, p)
	_ = s.Update(ctx, p)
	_ = s.Delete(ctx, p)
	_, _ = s.Get(ctx, "p", nil)
	_, _ = s.GetChilds(ctx, p)
}

// Test_Store_ReadWriteSplit verifies that Get and GetChilds count as reads and
// Add, Update and Delete as writes.
func Test_Store_ReadWriteSplit(t *testing.T) {
	var sw Switch
	st := sw.Enable()
	callAll(NewStore(&fakeStore{}, &sw))

	read, write := st.Read.Snapshot(), st.Write.Snapshot()
	assert.Equal(t, int64(2), read.Count)
	assert.Equal(t, int64(2), read.OK)
	assert.Equal(t, int64(3), write.Count)
	assert.Equal(t, int64(3), write.OK)
}

// Test_Store_Outcomes verifies how returned errors are classified, including
// wrapped errors.
func Test_Store_Outcomes(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		counter func(OpSnapshot) int64
	}{
		{"ok", nil, func(s OpSnapshot) int64 { return s.OK }},
		{"canceled", fmt.Errorf("wrapped: %w", context.Canceled), func(s OpSnapshot) int64 { return s.CtxErr }},
		{"deadline", context.DeadlineExceeded, func(s OpSnapshot) int64 { return s.CtxErr }},
		{"cidr conflict", fmt.Errorf("wrapped: %w", prefix.ErrCIDRConflict), func(s OpSnapshot) int64 { return s.CIDRConflict }},
		{"id conflict", fmt.Errorf("wrapped: %w", prefix.ErrPrimaryIDConflict), func(s OpSnapshot) int64 { return s.IDConflict }},
		{"other", errors.New("disk I/O error"), func(s OpSnapshot) int64 { return s.OtherErr }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sw Switch
			st := sw.Enable()
			callAll(NewStore(&fakeStore{err: tt.err}, &sw))

			read, write := st.Read.Snapshot(), st.Write.Snapshot()
			assert.Equal(t, int64(2), tt.counter(read))
			assert.Equal(t, int64(2), read.Count, "no other read outcome")
			assert.Equal(t, int64(3), tt.counter(write))
			assert.Equal(t, int64(3), write.Count, "no other write outcome")
		})
	}
}

// Test_Store_Disabled verifies that nothing is recorded while the Switch is
// disabled, that results pass through, and that Enable starts from zero.
func Test_Store_Disabled(t *testing.T) {
	var sw Switch
	errStore := errors.New("store error")
	s := NewStore(&fakeStore{err: errStore}, &sw)
	require.ErrorIs(t, s.Add(context.Background(), prefix.New("p", "10.0.0.0/24", nil)), errStore)

	st := sw.Enable()
	_, _ = s.Get(context.Background(), "p", nil)
	sw.Disable()
	callAll(s)

	assert.Equal(t, int64(1), st.Read.Snapshot().Count, "only the call while enabled")
	assert.Equal(t, int64(0), st.Write.Snapshot().Count)
	fresh := sw.Enable()
	assert.Equal(t, OpSnapshot{}, fresh.Read.Snapshot(), "re-enable starts from zero")
	assert.Equal(t, OpSnapshot{}, fresh.Write.Snapshot(), "re-enable starts from zero")
}

// Test_Store_NilSwitch verifies that a Store without a Switch passes calls
// through.
func Test_Store_NilSwitch(t *testing.T) {
	errStore := errors.New("store error")
	s := NewStore(&fakeStore{err: errStore}, nil)

	require.ErrorIs(t, s.Add(context.Background(), prefix.New("p", "10.0.0.0/24", nil)), errStore)
	_, err := s.Get(context.Background(), "p", nil)
	require.ErrorIs(t, err, errStore)
}

// Test_OpStats_Latency verifies bucket assignment, total time and the peak
// reset by Snapshot.
func Test_OpStats_Latency(t *testing.T) {
	durations := []time.Duration{
		0, 999 * time.Microsecond, // below 1ms
		time.Millisecond,             // below 10ms
		99 * time.Millisecond,        // below 100ms
		500 * time.Millisecond,       // below 1s
		time.Second, 3 * time.Second, // 1s or more
	}
	var o OpStats
	var total time.Duration
	for _, d := range durations {
		o.observe(d, nil)
		total += d
	}

	s := o.Snapshot()
	assert.Equal(t, Buckets{2, 1, 1, 1, 2}, s.Buckets)
	assert.Equal(t, int64(len(durations)), s.Count)
	assert.Equal(t, total, s.TotalTime)
	assert.Equal(t, 3*time.Second, s.PeakTime)

	next := o.Snapshot()
	assert.Equal(t, time.Duration(0), next.PeakTime)
	assert.Equal(t, s.TotalTime, next.TotalTime)
	assert.Equal(t, s.Buckets, next.Buckets)
}
