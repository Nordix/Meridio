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

package memory

import (
	"context"
	"fmt"
	"sync"
)

type MemoryIPAMStorage struct {
	prefixes map[string]map[string]struct{}
	mu       sync.Mutex
}

func NewStorage() *MemoryIPAMStorage {
	mis := &MemoryIPAMStorage{
		prefixes: make(map[string]map[string]struct{}),
	}
	return mis
}

func (mis *MemoryIPAMStorage) Add(ctx context.Context, prefix string, child string) error {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	_, exists := mis.prefixes[prefix]
	if !exists {
		mis.prefixes[prefix] = make(map[string]struct{})
	}
	_, childExists := mis.prefixes[prefix][child]
	if childExists {
		return fmt.Errorf("child %v already exists in %v", child, prefix)
	}
	mis.prefixes[prefix][child] = struct{}{}
	return nil
}

func (mis *MemoryIPAMStorage) Delete(ctx context.Context, prefix string, child string) error {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	_, exists := mis.prefixes[prefix]
	if !exists {
		return nil
	}
	delete(mis.prefixes[prefix], child)
	if len(mis.prefixes[prefix]) <= 0 {
		delete(mis.prefixes, prefix)
	}
	return nil
}

func (mis *MemoryIPAMStorage) Get(ctx context.Context, prefix string) ([]string, error) {
	mis.mu.Lock()
	defer mis.mu.Unlock()
	childsSlice := []string{}
	childs, exists := mis.prefixes[prefix]
	if !exists {
		return childsSlice, nil
	}
	for child := range childs {
		childsSlice = append(childsSlice, child)
	}
	return childsSlice, nil
}
