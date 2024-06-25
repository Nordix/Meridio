/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2024 OpenInfra Foundation Europe

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

package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
)

type MemoryIPAMStorage struct {
	prefixes []types.Prefix
	mu       sync.Mutex
}

func New() *MemoryIPAMStorage {
	mis := &MemoryIPAMStorage{
		prefixes: []types.Prefix{},
	}
	return mis
}

func (mis *MemoryIPAMStorage) Add(ctx context.Context, prefix types.Prefix) error {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	if prefix == nil {
		return nil
	}
	exists := mis.exists(prefix)
	if exists {
		return fmt.Errorf("prefix name %v already exists: %v", prefix.GetName(), prefix)
	}
	mis.prefixes = append(mis.prefixes, prefix)
	return nil
}

func (mis *MemoryIPAMStorage) Update(ctx context.Context, prefix types.Prefix) error {
	return nil // maintain baseline functionality
}

func (mis *MemoryIPAMStorage) Delete(ctx context.Context, prefix types.Prefix) error {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	return mis.delete(prefix)
}

func (mis *MemoryIPAMStorage) Get(ctx context.Context, name string, parent types.Prefix) (types.Prefix, error) {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	tempPrefix := prefix.New(name, "", parent)
	for _, p := range mis.prefixes {
		if tempPrefix.Equals(p) {
			return p, nil
		}
	}
	return nil, nil
}

func (mis *MemoryIPAMStorage) GetChilds(ctx context.Context, prefix types.Prefix) ([]types.Prefix, error) {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	return mis.getChilds(prefix), nil
}

func (mis *MemoryIPAMStorage) delete(prefix types.Prefix) error {
	if prefix == nil {
		return nil
	}
	var errFinal error
	for _, p := range mis.getChilds(prefix) {
		err := mis.delete(p)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}
	}
	index := mis.getIndex(prefix)
	mis.deleteIndex(index)
	return nil
}

func (mis *MemoryIPAMStorage) getChilds(prefix types.Prefix) []types.Prefix {
	childs := []types.Prefix{}
	for _, p := range mis.prefixes {
		if prefix.Equals(p.GetParent()) {
			childs = append(childs, p)
		}
	}
	return childs
}

func (mis *MemoryIPAMStorage) exists(prefix types.Prefix) bool {
	return mis.getIndex(prefix) >= 0
}

func (mis *MemoryIPAMStorage) getIndex(prefix types.Prefix) int {
	for i, p := range mis.prefixes {
		if prefix.Equals(p) {
			return i
		}
	}
	return -1
}

func (mis *MemoryIPAMStorage) deleteIndex(index int) {
	if index < 0 {
		return
	}
	mis.prefixes = append(mis.prefixes[:index], mis.prefixes[index+1:]...)
}
