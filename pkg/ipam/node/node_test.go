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

package node_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/node"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
)

func Test_Allocate(t *testing.T) {
	prefixNode := prefix.New("conduit-a", "172.0.0.0/24", nil)
	store := memory.New()
	node := node.New(prefixNode, store, types.NewPrefixLengths(20, 24, 32))
	assert.NotNil(t, node)
	prefix, err := node.Allocate(context.Background(), "bridge")
	assert.Nil(t, err)
	assert.NotNil(t, prefix)
	assert.True(t, prefix.GetParent().Equals(prefixNode))
	assert.Equal(t, prefix.GetCidr(), "172.0.0.1/32")
	prefix, err = node.Allocate(context.Background(), "bridge")
	assert.Nil(t, err)
	assert.NotNil(t, prefix)
	assert.True(t, prefix.GetParent().Equals(prefixNode))
	assert.Equal(t, prefix.GetCidr(), "172.0.0.1/32")
}
