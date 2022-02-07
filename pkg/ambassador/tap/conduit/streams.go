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

package conduit

import (
	"sync"

	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
)

const (
	closed = iota
	opened
)

type status int

type streamStatus struct {
	stream types.Stream
	status status
	mu     sync.Mutex
}

func (ss *streamStatus) setStatus(s status) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.status = s
}

type streamList struct {
	streams []*streamStatus
	mu      sync.Mutex
}

func newStreamList() *streamList {
	sl := &streamList{
		streams: []*streamStatus{},
	}
	return sl
}

func (sl *streamList) add(ss *streamStatus) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.streams = append(sl.streams, ss)
}

func (sl *streamList) del(strm *ambassadorAPI.Stream) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	for i, s := range sl.streams {
		equal := s.stream.Equals(strm)
		if equal {
			sl.streams = append(sl.streams[:i], sl.streams[i+1:]...)
			return
		}
	}
}

func (sl *streamList) get(strm *ambassadorAPI.Stream) *streamStatus {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	for _, s := range sl.streams {
		equal := s.stream.Equals(strm)
		if equal {
			return s
		}
	}
	return nil
}

func (sl *streamList) getList() []*streamStatus {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.streams
}
