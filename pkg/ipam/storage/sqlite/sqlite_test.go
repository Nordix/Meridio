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

package sqlite_test

import (
	"context"
	"os"
	"testing"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/stretchr/testify/assert"
)

const dbFileName = "test.db"

func Test_Add_Get(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer func() {
		_ = os.Remove(dbFileName)
	}()

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	assert.NotNil(t, store)
	err = store.Add(context.Background(), p1)
	assert.Nil(t, err)
	pGet, err := store.Get(context.Background(), "abc", nil)
	assert.Nil(t, err)
	assert.Equal(t, p1, pGet)

	p2 := prefix.New("abc", "192.168.0.0/28", p1)
	err = store.Add(context.Background(), p2)
	assert.Nil(t, err)
	pGet, err = store.Get(context.Background(), "abc", nil)
	assert.Nil(t, err)
	assert.Equal(t, p1, pGet)
	pGet, err = store.Get(context.Background(), "abc", p1)
	assert.Nil(t, err)
	assert.Equal(t, p2, pGet)

	p3 := prefix.New("def", "192.168.0.0/30", p2)
	err = store.Add(context.Background(), p3)
	assert.Nil(t, err)
	pGet, err = store.Get(context.Background(), "def", p2)
	assert.Nil(t, err)
	assert.Equal(t, p3, pGet)

	p4 := prefix.New("ghi", "192.168.0.0/32", p3)
	err = store.Add(context.Background(), p4)
	assert.Nil(t, err)
	pGet, err = store.Get(context.Background(), "ghi", p3)
	assert.Nil(t, err)
	assert.Equal(t, p4, pGet)

	p5 := prefix.New("abc", "192.168.0.1/32", nil)
	err = store.Add(context.Background(), p5)
	assert.NotNil(t, err)
}

func Test_GetChilds(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer func() {
		_ = os.Remove(dbFileName)
	}()

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/16", nil)
	_ = store.Add(context.Background(), p1)
	p2 := prefix.New("abc", "192.168.0.0/24", p1)
	_ = store.Add(context.Background(), p2)
	p3 := prefix.New("def", "192.168.0.0/24", p1)
	_ = store.Add(context.Background(), p3)
	p4 := prefix.New("def", "192.168.0.0/32", p3)
	_ = store.Add(context.Background(), p4)

	childs, err := store.GetChilds(context.Background(), p1)
	assert.Nil(t, err)
	assert.Len(t, childs, 2)
	assert.Contains(t, childs, p2)
	assert.Contains(t, childs, p3)

	childs, err = store.GetChilds(context.Background(), p2)
	assert.Nil(t, err)
	assert.Len(t, childs, 0)

	childs, err = store.GetChilds(context.Background(), p3)
	assert.Nil(t, err)
	assert.Len(t, childs, 1)
	assert.Contains(t, childs, p4)
}

func Test_Delete(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer func() {
		_ = os.Remove(dbFileName)
	}()

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	_ = store.Add(context.Background(), p1)
	p2 := prefix.New("abc", "192.168.0.0/32", p1)
	_ = store.Add(context.Background(), p2)
	p3 := prefix.New("def", "192.168.0.1/32", p1)
	_ = store.Add(context.Background(), p3)
	err = store.Delete(context.Background(), p2)
	assert.Nil(t, err)
	childs, _ := store.GetChilds(context.Background(), p1)
	assert.Len(t, childs, 1)
	assert.Contains(t, childs, p3)

	err = store.Delete(context.Background(), p1)
	assert.Nil(t, err)
	pGet, _ := store.Get(context.Background(), "abc", nil)
	assert.Nil(t, pGet)
	pGet, _ = store.Get(context.Background(), "def", p1)
	assert.Nil(t, pGet)
}
