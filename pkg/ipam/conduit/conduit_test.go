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

package conduit_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/conduit"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
)

func Test_GetNode(t *testing.T) {
	prefixConduit := prefix.New("conduit-a", "172.0.0.0/20", nil)
	store := memory.New()
	conduit := conduit.New(prefixConduit, store, types.NewPrefixLengths(20, 24, 32))
	assert.NotNil(t, conduit)
	node, err := conduit.GetNode(context.Background(), "node-a")
	assert.Nil(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.GetParent().Equals(prefixConduit))
	assert.Equal(t, node.GetName(), "node-a")
	assert.Equal(t, node.GetCidr(), "172.0.0.0/24")
	node, err = conduit.GetNode(context.Background(), "node-b")
	assert.Nil(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.GetParent().Equals(prefixConduit))
	assert.Equal(t, node.GetName(), "node-b")
	assert.Equal(t, node.GetCidr(), "172.0.1.0/24")
	node, err = conduit.GetNode(context.Background(), "node-a")
	assert.Nil(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.GetParent().Equals(prefixConduit))
	assert.Equal(t, node.GetName(), "node-a")
	assert.Equal(t, node.GetCidr(), "172.0.0.0/24")
}

func Test_RemoveNode(t *testing.T) {
	prefixConduit := prefix.New("conduit-a", "172.0.0.0/20", nil)
	store := memory.New()
	conduit := conduit.New(prefixConduit, store, types.NewPrefixLengths(20, 24, 32))
	assert.NotNil(t, conduit)
	node, err := conduit.GetNode(context.Background(), "node-a")
	assert.Nil(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.GetParent().Equals(prefixConduit))
	assert.Equal(t, node.GetName(), "node-a")
	assert.Equal(t, node.GetCidr(), "172.0.0.0/24")
	err = conduit.RemoveNode(context.Background(), "node-a")
	assert.Nil(t, err)
	node, err = conduit.GetNode(context.Background(), "node-b")
	assert.Nil(t, err)
	assert.NotNil(t, node)
	assert.True(t, node.GetParent().Equals(prefixConduit))
	assert.Equal(t, node.GetName(), "node-b")
	assert.Equal(t, node.GetCidr(), "172.0.0.0/24")
}
