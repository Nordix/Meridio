/*
Copyright (c) 2021-2022 Nordix Foundation

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

package registry

import (
	"context"
	"sync"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
)

type Registry struct {
	mu       sync.Mutex
	streams  []*ambassadorAPI.StreamStatus
	watchers map[*RegistryWatcher]struct{}
}

func New() *Registry {
	r := &Registry{
		streams:  []*ambassadorAPI.StreamStatus{},
		watchers: make(map[*RegistryWatcher]struct{}),
	}
	return r
}

func (r *Registry) SetStatus(stream *ambassadorAPI.Stream, status ambassadorAPI.StreamStatus_Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	strmStatus := r.getStream(stream)
	if strmStatus == nil {
		strmStatus = &ambassadorAPI.StreamStatus{
			Status: status,
			Stream: stream,
		}
		r.streams = append(r.streams, strmStatus)
	}
	strmStatus.Status = status
	r.notifyAllWatchers()
}

func (r *Registry) Remove(stream *ambassadorAPI.Stream) {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := r.getStreamIndex(stream)
	if index < 0 {
		return
	}
	r.streams = append(r.streams[:index], r.streams[index+1:]...)
	r.notifyAllWatchers()
}

func (r *Registry) Watch(ctx context.Context, stream *ambassadorAPI.Stream) (types.Watcher, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	watcher := NewRegistryWatcher(stream)
	r.watchers[watcher] = struct{}{}
	watcher.Notify(r.copy())
	return watcher, nil
}

func (r *Registry) getStreamIndex(stream *ambassadorAPI.Stream) int {
	for i, s := range r.streams {
		if stream.Equals(s.GetStream()) {
			return i
		}
	}
	return -1
}

func (r *Registry) getStream(stream *ambassadorAPI.Stream) *ambassadorAPI.StreamStatus {
	index := r.getStreamIndex(stream)
	if index < 0 {
		return nil
	}
	return r.streams[index]
}

func (r *Registry) notifyAllWatchers() {
	for watcher := range r.watchers {
		if watcher.IsStopped() {
			delete(r.watchers, watcher)
		}
		watcher.Notify(r.copy())
	}
}

// todo: this copy is used to avoid data races when notifying the changes
// if the status change while the watcher reads the status, the data race
// could happen
func (r *Registry) copy() []*ambassadorAPI.StreamStatus {
	streams := []*ambassadorAPI.StreamStatus{}
	for _, stream := range r.streams {
		ns := &ambassadorAPI.StreamStatus{
			Stream: stream.GetStream(),
			Status: stream.GetStatus(),
		}
		streams = append(streams, ns)
	}
	return streams
}
