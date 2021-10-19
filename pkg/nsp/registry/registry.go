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
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/types"
)

type targetKey struct {
	Ips    []string
	Stream *nspAPI.Stream
	Type   nspAPI.Target_Type
}

type TargetRegistryMemory struct {
	targets []*nspAPI.Target
	Chan    chan<- struct{}
	mu      sync.Mutex
}

func New(c chan<- struct{}) types.TargetRegistry {
	targetRegistryMemory := &TargetRegistryMemory{
		Chan:    c,
		targets: []*nspAPI.Target{},
	}
	return targetRegistryMemory
}

func (trm *TargetRegistryMemory) Set(target *nspAPI.Target) {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	index := trm.getIndex(target)
	if index == -1 { // add
		trm.targets = append(trm.targets, target)
		trm.notify()
		return
	}
	//update
	trm.targets[index].Context = target.GetContext()
	trm.targets[index].Status = target.GetStatus()
	trm.notify()
}

func (trm *TargetRegistryMemory) Remove(target *nspAPI.Target) {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	index := trm.getIndex(target)
	if index == -1 {
		return
	}
	trm.targets = append(trm.targets[:index], trm.targets[index+1:]...)
	trm.notify()
}

func (trm *TargetRegistryMemory) Get(target *nspAPI.Target) []*nspAPI.Target {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	if target == nil {
		return trm.targets
	}
	targets := []*nspAPI.Target{}
	for _, t := range trm.targets {
		if target.Equals(t) {
			targets = append(targets, t)
		}
	}
	return targets
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

func (trm *TargetRegistryMemory) notify() {
	if trm.Chan == nil {
		return
	}
	trm.Chan <- struct{}{}
}
