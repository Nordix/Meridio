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

package prefix_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/stretchr/testify/assert"
)

func Test_Prefix_IPv4_Allocate(t *testing.T) {
	p := prefix.New("parent", "169.16.0.0/16", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 24, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "169.16.0.0/24", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 24, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "169.16.1.0/24", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-c", 32, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-c", newPrefix.GetName())
	assert.Equal(t, "169.16.2.0/32", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-d", 32, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-d", newPrefix.GetName())
	assert.Equal(t, "169.16.2.1/32", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-e", 17, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-e", newPrefix.GetName())
	assert.Equal(t, "169.16.128.0/17", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
}

func Test_Prefix_IPv4_Allocate_Full(t *testing.T) {
	p := prefix.New("parent", "169.16.0.0/24", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 25, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "169.16.0.0/25", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 25, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "169.16.0.128/25", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	_, err = prefix.Allocate(context.TODO(), p, "child-b", 32, store)
	assert.NotNil(t, err)
}

func Test_Prefix_IPv6_Allocate(t *testing.T) {
	p := prefix.New("parent", "2001:1::/32", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 64, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "2001:1::/64", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 64, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:1::/64", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-c", 128, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-c", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:2::/128", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-d", 128, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-d", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:2::1/128", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-e", 33, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-e", newPrefix.GetName())
	assert.Equal(t, "2001:1:8000::/33", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
}

func Test_Prefix_IPv6_Allocate_Full(t *testing.T) {
	p := prefix.New("parent", "2001:1::/64", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 65, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "2001:1::/65", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 65, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:0:8000::/65", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	_, err = prefix.Allocate(context.TODO(), p, "child-c", 128, store)
	assert.NotNil(t, err)
}
