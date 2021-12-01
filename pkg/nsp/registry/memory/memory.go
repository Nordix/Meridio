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

package registry

import (
	"context"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry/common"
	"github.com/nordix/meridio/pkg/nsp/types"
)

type targetKey struct {
	Ips    []string
	Stream *nspAPI.Stream
	Type   nspAPI.Target_Type
}

type TargetRegistryMemory struct {
	targets  []*nspAPI.Target
	mu       sync.Mutex
	watchers map[*common.RegistryWatcher]struct{}
}

func New() types.TargetRegistry {
	targetRegistryMemory := &TargetRegistryMemory{
		targets: []*nspAPI.Target{},
	}
	return targetRegistryMemory
}

func (trm *TargetRegistryMemory) Set(ctx context.Context, target *nspAPI.Target) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	index := trm.getIndex(target)
	if index == -1 { // add
		trm.targets = append(trm.targets, target)
		trm.notifyAllWatchers()
		return nil
	}
	//update
	trm.targets[index].Context = target.GetContext()
	trm.targets[index].Status = target.GetStatus()
	trm.notifyAllWatchers()
	return nil
}

func (trm *TargetRegistryMemory) Remove(ctx context.Context, target *nspAPI.Target) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	index := trm.getIndex(target)
	if index == -1 {
		return nil
	}
	trm.targets = append(trm.targets[:index], trm.targets[index+1:]...)
	trm.notifyAllWatchers()
	return nil
}

func (trm *TargetRegistryMemory) Watch(ctx context.Context, target *nspAPI.Target) (types.TargetWatcher, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	trm.setWatchersIfNil()
	watcher := common.NewRegistryWatcher(target)
	trm.watchers[watcher] = struct{}{}
	watcher.Notify(trm.targets)
	return watcher, nil
}

func (trm *TargetRegistryMemory) Get(ctx context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	return common.Filter(target, trm.targets), nil
}

func (trm *TargetRegistryMemory) getIndex(target *nspAPI.Target) int {
	targetKey := getTargetKey(target)
	for i, t := range trm.targets {
		tk := getTargetKey(t)
		if targetKey.Equals(tk) {
			return i
		}
	}
	return -1
}

func getTargetKey(target *nspAPI.Target) *targetKey {
	tk := &targetKey{
		Ips:    target.GetIps(),
		Stream: target.GetStream(),
		Type:   target.GetType(),
	}
	return tk
}

func (tk *targetKey) Equals(tk2 *targetKey) bool {
	if tk == nil || tk2 == nil {
		return false
	}
	stream := true
	if tk.Stream != nil && tk2.Stream != nil {
		stream = tk.Stream.Equals(tk2.Stream)
	}
	return CompareIps(tk.Ips, tk2.Ips) && stream && tk.Type == tk2.Type
}

func CompareIps(ips1 []string, ips2 []string) bool {
	if len(ips1) != len(ips2) {
		return false
	}
	ips1Map := make(map[string]struct{})
	for _, ip := range ips1 {
		ips1Map[ip] = struct{}{}
	}
	for _, ip := range ips2 {
		_, exists := ips1Map[ip]
		if !exists {
			return false
		}
	}
	return true
}

func (trm *TargetRegistryMemory) notifyAllWatchers() {
	trm.setWatchersIfNil()
	for watcher := range trm.watchers {
		if watcher.IsStopped() {
			delete(trm.watchers, watcher)
		}
		watcher.Notify(trm.targets)
	}
}

func (trm *TargetRegistryMemory) setWatchersIfNil() {
	if trm.watchers == nil {
		trm.watchers = make(map[*common.RegistryWatcher]struct{})
	}
}
