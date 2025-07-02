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
	"fmt"

	"github.com/nordix/meridio/pkg/ipam/node"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
)

type Conduit struct {
	types.Prefix
	Store         types.Storage
	PrefixLengths *types.PrefixLengths
}

// New is the constructor for the Conduit struct
// prefix - prefix of the conduit
// store - storage for the prefix and its childs (nodes)
// prefixLengths - prefix length used to allocate the childs (nodes)
func New(prefix types.Prefix, store types.Storage, prefixLengths *types.PrefixLengths) *Conduit {
	p := &Conduit{
		Prefix:        prefix,
		Store:         store,
		PrefixLengths: prefixLengths,
	}
	return p
}

// GetNode returns the node with the name in parameter and with as parent the current conduit.
// If not existing, a new one will be created and returned.
func (c *Conduit) GetNode(ctx context.Context, name string) (types.Node, error) {
	p, err := c.Store.Get(ctx, name, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get node (%s) from store in conduit prefix (%s): %w", name, c.GetName(), err)
	}
	var n types.Node
	if p == nil {
		n, err = c.addNode(ctx, name)
		if err != nil {
			return nil, err
		}
	} else {
		// Refresh the entry's updatedAt timestamp in storage
		// TODO: Would be nice if frequent updates could be avoided (e.g. based on UpdatedAt time)
		if err = c.Store.Update(ctx, p); err != nil {
			return nil, fmt.Errorf("failed to update node (%s) in conduit prefix (%s): %w", name, c.GetName(), err)
		}
		n = node.New(p, c.Store, c.PrefixLengths)
	}
	return n, nil
}

// RemoveNode removes the node with the name in parameter and with as parent the current conduit.
// no error is returned if the node does not exist.
func (c *Conduit) RemoveNode(ctx context.Context, name string) error {
	prefix, err := c.Store.Get(ctx, name, c)
	if err != nil {
		return fmt.Errorf("failed to get node (%s) from store while removing node in conduit prefix (%s): %w", name, c.GetName(), err)
	}
	err = c.Store.Delete(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to delete (%s) from store in conduit prefix (%s): %w", name, c.GetName(), err)
	}
	return nil
}

func (c *Conduit) addNode(ctx context.Context, name string) (types.Node, error) {
	newPrefix, err := prefix.Allocate(ctx, c, name, c.PrefixLengths.NodeLength, c.Store)
	if err != nil {
		return nil, fmt.Errorf("failed to add node (%s) in conduit prefix (%s): %w", name, c.GetName(), err)
	}
	return node.New(newPrefix, c.Store, c.PrefixLengths), nil
}
