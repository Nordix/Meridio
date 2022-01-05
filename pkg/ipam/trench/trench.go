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

package trench

import (
	"context"

	"github.com/nordix/meridio/pkg/ipam/conduit"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
)

type Trench struct {
	types.Prefix
	Store          types.Storage
	ConduitWatcher *ConduitWatcher
	PrefixLengths  *types.PrefixLengths
}

func New(ctx context.Context, prefix types.Prefix, store types.Storage, prefixLengths *types.PrefixLengths) (*Trench, error) {
	p, err := store.Get(ctx, prefix.GetName(), nil)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = prefix
		err = store.Add(ctx, prefix)
		if err != nil {
			return nil, err
		}
	}
	t := &Trench{
		Prefix:        p,
		Store:         store,
		PrefixLengths: prefixLengths,
	}
	return t, nil
}

func (t *Trench) GetConduit(ctx context.Context, name string) (types.Conduit, error) {
	prefix, err := t.Store.Get(ctx, name, t)
	if err != nil {
		return nil, err
	}
	if prefix == nil {
		return nil, nil
	}
	return conduit.New(prefix, t.Store, t.PrefixLengths), nil
}

func (t *Trench) AddConduit(ctx context.Context, name string) (types.Conduit, error) {
	newPrefix, err := prefix.Allocate(ctx, t, name, t.PrefixLengths.ConduitLength, t.Store)
	if err != nil {
		return nil, err
	}
	return conduit.New(newPrefix, t.Store, t.PrefixLengths), nil
}

func (t *Trench) RemoveConduit(ctx context.Context, name string) error {
	prefix, err := t.Store.Get(ctx, name, t)
	if err != nil {
		return err
	}
	return t.Store.Delete(ctx, prefix)
}
