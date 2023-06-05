/*
Copyright (c) 2023 Nordix Foundation

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

package interfacename_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/stretchr/testify/require"
)

const PREFIX = "dev"

type dummyGenerator struct {
	mu      sync.Mutex
	counter int
}

func (dg *dummyGenerator) Generate(prefix string, maxLength int) string {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	dg.counter++
	return fmt.Sprintf("%s-%d", prefix, dg.counter)
}

func (dg *dummyGenerator) Release(name string) {
}

var CacheGenerateTests = []struct {
	id       string
	expected string
}{
	{
		id:       "first-id",
		expected: fmt.Sprintf("%s-1", PREFIX),
	},
	{
		id:       "second-id",
		expected: fmt.Sprintf("%s-2", PREFIX),
	},
	{
		id:       "first-id",
		expected: fmt.Sprintf("%s-1", PREFIX),
	},
	{
		id:       "second-id",
		expected: fmt.Sprintf("%s-2", PREFIX),
	},
}

var CacheExpireTests = []struct {
	id       string
	expected string
}{
	{
		id:       "first-id",
		expected: fmt.Sprintf("%s-1", PREFIX),
	},
	{
		id:       "second-id",
		expected: fmt.Sprintf("%s-2", PREFIX),
	},
	{
		id:       "first-id",
		expected: fmt.Sprintf("%s-3", PREFIX),
	},
	{
		id:       "second-id",
		expected: fmt.Sprintf("%s-4", PREFIX),
	},
}

func TestCacheGenerate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cache := interfacename.NewInterfaceNameChache(ctx, &dummyGenerator{})
	for _, nt := range CacheGenerateTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}
}

func TestCacheRecover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cache := interfacename.NewInterfaceNameChache(ctx, &dummyGenerator{})
	for _, nt := range CacheGenerateTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
		<-time.After(10 * time.Microsecond)
	}
}

func TestCacheImmediateExpire(t *testing.T) {
	instantReleaseTrigger := func() <-chan struct{} {
		channel := make(chan struct{}, 1)
		channel <- struct{}{}
		return channel
	}

	cache := interfacename.NewInterfaceNameChache(
		context.TODO(),
		&dummyGenerator{},
		interfacename.WithReleaseTrigger(instantReleaseTrigger),
	)

	for _, nt := range CacheExpireTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
	}
}

func TestCacheExpire(t *testing.T) {
	instantReleaseTrigger := func() <-chan struct{} {
		channel := make(chan struct{}, 1)
		go func() {
			<-time.After(1 * time.Millisecond)
			channel <- struct{}{}
			close(channel)
		}()
		return channel
	}

	cache := interfacename.NewInterfaceNameChache(
		context.TODO(),
		&dummyGenerator{},
		interfacename.WithReleaseTrigger(instantReleaseTrigger),
	)

	for _, nt := range CacheExpireTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
		<-time.After(10 * time.Millisecond)
	}
}
