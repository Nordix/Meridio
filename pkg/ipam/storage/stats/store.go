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
	"time"

	"github.com/nordix/meridio/pkg/ipam/types"
)

// Store decorates a types.Storage and records its operations while the Switch
// is enabled. The latency of an operation includes any wait for a lock inside
// the decorated store. While the Switch is disabled or nil, calls pass
// straight through.
type Store struct {
	store types.Storage
	sw    *Switch
}

// NewStore returns store decorated with statistics controlled by sw.
func NewStore(store types.Storage, sw *Switch) *Store {
	return &Store{store: store, sw: sw}
}

func (s *Store) Add(ctx context.Context, p types.Prefix) error {
	st := s.sw.Stats()
	if st == nil {
		return s.store.Add(ctx, p)
	}
	start := time.Now()
	err := s.store.Add(ctx, p)
	st.Write.observe(time.Since(start), err)
	return err
}

func (s *Store) Update(ctx context.Context, p types.Prefix) error {
	st := s.sw.Stats()
	if st == nil {
		return s.store.Update(ctx, p)
	}
	start := time.Now()
	err := s.store.Update(ctx, p)
	st.Write.observe(time.Since(start), err)
	return err
}

func (s *Store) Delete(ctx context.Context, p types.Prefix) error {
	st := s.sw.Stats()
	if st == nil {
		return s.store.Delete(ctx, p)
	}
	start := time.Now()
	err := s.store.Delete(ctx, p)
	st.Write.observe(time.Since(start), err)
	return err
}

func (s *Store) Get(ctx context.Context, name string, parent types.Prefix) (types.Prefix, error) {
	st := s.sw.Stats()
	if st == nil {
		return s.store.Get(ctx, name, parent)
	}
	start := time.Now()
	p, err := s.store.Get(ctx, name, parent)
	st.Read.observe(time.Since(start), err)
	return p, err
}

func (s *Store) GetChilds(ctx context.Context, p types.Prefix) ([]types.Prefix, error) {
	st := s.sw.Stats()
	if st == nil {
		return s.store.GetChilds(ctx, p)
	}
	start := time.Now()
	childs, err := s.store.GetChilds(ctx, p)
	st.Read.observe(time.Since(start), err)
	return childs, err
}
