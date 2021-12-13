/*
Copyright (c) 2021 Nordix Foundation

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

package common

import (
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

const (
	channelBufferSize = 1
)

type RegistryWatcher struct {
	targetSelector *nspAPI.Target
	c              chan []*nspAPI.Target
	stopped        bool
	mu             sync.Mutex
}

func NewRegistryWatcher(target *nspAPI.Target) *RegistryWatcher {
	rw := &RegistryWatcher{
		targetSelector: target,
		c:              make(chan []*nspAPI.Target, channelBufferSize),
		stopped:        false,
	}
	return rw
}

func (rw *RegistryWatcher) Stop() {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	rw.stopped = true
	close(rw.c)
}

func (rw *RegistryWatcher) ResultChan() <-chan []*nspAPI.Target {
	if rw.stopped {
		return nil
	}
	return rw.c
}

func (rw *RegistryWatcher) IsStopped() bool {
	return rw.stopped
}

func (rw *RegistryWatcher) Notify(targets []*nspAPI.Target) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.IsStopped() {
		return
	}
	result := Filter(rw.targetSelector, targets)
	// todo: cache (to not send same target list 2 times)
	select {
	case <-rw.c:
	default:
	}
	rw.c <- result
}
