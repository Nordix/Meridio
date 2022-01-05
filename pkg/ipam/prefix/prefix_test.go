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
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	assert.NotNil(t, p1)
	assert.Equal(t, p1.GetName(), "abc")
	assert.Equal(t, p1.GetCidr(), "192.168.0.0/24")
	assert.Equal(t, p1.GetParent(), nil)
	p2 := prefix.New("def", "192.168.0.0/32", p1)
	assert.NotNil(t, p2)
	assert.Equal(t, p2.GetName(), "def")
	assert.Equal(t, p2.GetCidr(), "192.168.0.0/32")
	assert.Equal(t, p2.GetParent(), p1)
}

func Test_Equals(t *testing.T) {
	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	p2 := prefix.New("def", "192.168.0.0/32", p1)
	assert.True(t, p1.Equals(p1))
	assert.True(t, p2.Equals(p2))
	assert.False(t, p1.Equals(p2))
	assert.False(t, p2.Equals(p1))
	assert.True(t, p2.GetParent().Equals(p1))
	assert.True(t, p1.Equals(p2.GetParent()))
	p3 := prefix.New("abc", "192.168.0.0/24", nil)
	assert.True(t, p1.Equals(p3))
	p4 := prefix.New("def", "192.168.0.0/32", p1)
	assert.True(t, p2.Equals(p4))
	p5 := prefix.New("abc1", "192.168.0.0/24", nil)
	assert.False(t, p1.Equals(p5))
}
