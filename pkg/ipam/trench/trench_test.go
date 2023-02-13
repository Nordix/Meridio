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

package trench_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/nordix/meridio/pkg/ipam/trench"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
)

func Test_Add_Get_Delete_Conduit(t *testing.T) {
	prefixTrench := prefix.New("trench-a", "172.0.0.0/16", nil)
	store := memory.New()
	// trench := trench.New(prefixTrench, store, types.NewPrefixLengths(20, 24, 32))
	trench := &trench.Trench{
		Prefix:        prefixTrench,
		Store:         store,
		PrefixLengths: types.NewPrefixLengths(20, 24, 32),
	}
	assert.NotNil(t, trench)
	conduit, err := trench.GetConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	assert.Nil(t, conduit)
	conduit, err = trench.AddConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	assert.NotNil(t, conduit)
	assert.True(t, conduit.GetParent().Equals(prefixTrench))
	assert.Equal(t, conduit.GetName(), "conduit-a")
	assert.Equal(t, conduit.GetCidr(), "172.0.0.0/20")
	conduit, err = trench.GetConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	assert.NotNil(t, conduit)
	assert.True(t, conduit.GetParent().Equals(prefixTrench))
	assert.Equal(t, conduit.GetName(), "conduit-a")
	assert.Equal(t, conduit.GetCidr(), "172.0.0.0/20")
	err = trench.RemoveConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	conduit, err = trench.GetConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	assert.Nil(t, conduit)
}

func Test_Add_Conduit_Full(t *testing.T) {
	prefixTrench := prefix.New("trench-a", "172.0.0.0/31", nil)
	store := memory.New()
	trench := &trench.Trench{
		Prefix:        prefixTrench,
		Store:         store,
		PrefixLengths: types.NewPrefixLengths(32, 32, 32),
	}
	assert.NotNil(t, trench)
	conduit, err := trench.GetConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	assert.Nil(t, conduit)
	conduit, err = trench.AddConduit(context.Background(), "conduit-a")
	assert.Nil(t, err)
	assert.NotNil(t, conduit)
	assert.True(t, conduit.GetParent().Equals(prefixTrench))
	assert.Equal(t, conduit.GetName(), "conduit-a")
	assert.Equal(t, conduit.GetCidr(), "172.0.0.0/32")
	conduit, err = trench.AddConduit(context.Background(), "conduit-b")
	assert.Nil(t, err)
	assert.NotNil(t, conduit)
	assert.True(t, conduit.GetParent().Equals(prefixTrench))
	assert.Equal(t, conduit.GetName(), "conduit-b")
	assert.Equal(t, conduit.GetCidr(), "172.0.0.1/32")
	conduit, err = trench.AddConduit(context.Background(), "conduit-c")
	assert.NotNil(t, err)
	assert.Nil(t, conduit)
}
