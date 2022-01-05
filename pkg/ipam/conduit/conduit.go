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

package conduit

import (
	"context"

	"github.com/nordix/meridio/pkg/ipam/node"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
)

type Conduit struct {
	types.Prefix
	Store         types.Storage
	PrefixLengths *types.PrefixLengths
}

func New(prefix types.Prefix, store types.Storage, prefixLengths *types.PrefixLengths) *Conduit {
	p := &Conduit{
		Prefix:        prefix,
		Store:         store,
		PrefixLengths: prefixLengths,
	}
	return p
}

func (c *Conduit) GetNode(ctx context.Context, name string) (types.Node, error) {
	p, err := c.Store.Get(ctx, name, c)
	if err != nil {
		return nil, err
	}
	var n types.Node
	if p == nil {
		n, err = c.addNode(ctx, name)
		if err != nil {
			return nil, err
		}
	} else {
		n = node.New(p, c.Store, c.PrefixLengths)
	}
	return n, nil
}

func (c *Conduit) RemoveNode(ctx context.Context, name string) error {
	prefix, err := c.Store.Get(ctx, name, c)
	if err != nil {
		return err
	}
	return c.Store.Delete(ctx, prefix)
}

func (c *Conduit) addNode(ctx context.Context, name string) (types.Node, error) {
	newPrefix, err := prefix.Allocate(ctx, c, name, c.PrefixLengths.NodeLength, c.Store)
	if err != nil {
		return nil, err
	}
	return node.New(newPrefix, c.Store, c.PrefixLengths), nil
}
