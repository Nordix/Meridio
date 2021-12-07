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

package ipam_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/stretchr/testify/assert"
)

func Test_Prefix_IPv4_Allocate(t *testing.T) {
	p, err := ipam.NewPrefix("169.16.0.0/16")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 24)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.0/24", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 24)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.1.0/24", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.2.0/32", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.2.1/32", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 17)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.128.0/17", newPrefix)
}

func Test_Prefix_IPv4_Allocate_Full(t *testing.T) {
	p, err := ipam.NewPrefix("169.16.0.0/24")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 25)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.0/25", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 25)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.128/25", newPrefix)
	_, err = p.Allocate(context.TODO(), 32)
	assert.NotNil(t, err)
}

func Test_Prefix_IPv4_Release(t *testing.T) {
	p, err := ipam.NewPrefix("169.16.0.0/24")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.0/32", newPrefix)
	err = p.Release(context.TODO(), "169.16.0.0/32")
	assert.Nil(t, err)
	newPrefix, err = p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.0/32", newPrefix)
}

func Test_Prefix_IPv6_Allocate(t *testing.T) {
	p, err := ipam.NewPrefix("2001:1::/32")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 64)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1::/64", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 64)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1:0:1::/64", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 128)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1:0:2::/128", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 128)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1:0:2::1/128", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 33)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1:8000::/33", newPrefix)
}

func Test_Prefix_IPv6_Allocate_Full(t *testing.T) {
	p, err := ipam.NewPrefix("2001:1::/64")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 65)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1::/65", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 65)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1:0:0:8000::/65", newPrefix)
	_, err = p.Allocate(context.TODO(), 128)
	assert.NotNil(t, err)
}

func Test_Prefix_IPv6_Release(t *testing.T) {
	p, err := ipam.NewPrefix("2001:1::/64")
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 128)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1::/128", newPrefix)
	err = p.Release(context.TODO(), "2001:1::/128")
	assert.Nil(t, err)
	newPrefix, err = p.Allocate(context.TODO(), 128)
	assert.Nil(t, err)
	assert.Equal(t, "2001:1::/128", newPrefix)
}

func Test_Prefix_WithStorage(t *testing.T) {
	store := memory.NewStorage()

	p, err := ipam.NewPrefixWithStorage("169.16.0.0/24", store)
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err := p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.0/32", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.1/32", newPrefix)

	p, err = ipam.NewPrefixWithStorage("169.16.0.0/24", store)
	assert.Nil(t, err)
	assert.NotNil(t, p)
	newPrefix, err = p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.2/32", newPrefix)
	newPrefix, err = p.Allocate(context.TODO(), 32)
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.3/32", newPrefix)
}
