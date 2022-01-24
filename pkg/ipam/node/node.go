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

package node

import (
	"context"
	"fmt"
	"net"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
)

type Node struct {
	types.Prefix
	Store         types.Storage
	PrefixLengths *types.PrefixLengths
}

// New is the constructor for the Node struct
// prefix - prefix of the Node
// store - storage for the prefix and its childs (target, LB, Bridge...)
// prefixLengths - prefix length used to allocate the childs (target, LB, Bridge...)
func New(prefix types.Prefix, store types.Storage, prefixLengths *types.PrefixLengths) *Node {
	p := &Node{
		Prefix:        prefix,
		Store:         store,
		PrefixLengths: prefixLengths,
	}
	return p
}

// Allocate returns the prefix with the name in parameter and with as parent the current node.
// If not existing, a new one will be created and returned.
func (n *Node) Allocate(ctx context.Context, name string) (types.Prefix, error) {
	p, err := n.Store.Get(ctx, name, n)
	if err != nil {
		return nil, err
	}
	if p == nil {
		blocklist, err := n.getBlocklist()
		if err != nil {
			return nil, err
		}
		p, err = prefix.AllocateWithBlocklist(ctx, n, name, n.PrefixLengths.ChildLength, n.Store, blocklist)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Release removes the prefix with the name in parameter and with as parent the current node.
// no error is returned if the prefix does not exist.
func (n *Node) Release(ctx context.Context, name string) error {
	prefix, err := n.Store.Get(ctx, name, n)
	if err != nil {
		return err
	}
	return n.Store.Delete(ctx, prefix)
}

func (n *Node) getBlocklist() ([]string, error) {
	_, ipNet, err := net.ParseCIDR(n.GetCidr())
	if err != nil {
		return nil, err
	}
	first := fmt.Sprintf("%s/%d", ipNet.IP.String(), n.PrefixLengths.ChildLength)
	last := fmt.Sprintf("%s/%d", prefix.LastIP(ipNet), n.PrefixLengths.ChildLength)
	return []string{first, last}, nil
}
