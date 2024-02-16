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
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

const PREFIX = "dev"

func newDummyGenerator() *dummyGenerator {
	return &dummyGenerator{
		usedNames: map[string]struct{}{},
	}
}

type dummyGenerator struct {
	usedNames map[string]struct{}
	mu        sync.Mutex
	counter   int
}

func (dg *dummyGenerator) Generate(prefix string, maxLength int) string {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	dg.counter++
	name := fmt.Sprintf("%s-%d", prefix, dg.counter)
	dg.usedNames[name] = struct{}{}
	return name
}

func (dg *dummyGenerator) Release(name string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()
	delete(dg.usedNames, name)
}

func (dg *dummyGenerator) Reserve(name, prefix string, maxLength int) error {
	if len(name) > maxLength || len(name) == len(prefix) || (prefix != "" && !strings.HasPrefix(name, prefix)) {
		return fmt.Errorf("wrong name format")
	}

	if _, ok := dg.usedNames[name]; ok {
		// already taken
		return os.ErrExist
	}
	dg.usedNames[name] = struct{}{}

	return nil
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

var CacheCancelRelease = []struct {
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
}

func TestCacheGenerate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cache := interfacename.NewInterfaceNameChache(ctx, newDummyGenerator())
	for _, nt := range CacheGenerateTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}
}

func TestCacheRecover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cache := interfacename.NewInterfaceNameChache(ctx, newDummyGenerator())
	for _, nt := range CacheGenerateTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
		<-time.After(10 * time.Microsecond)
	}
}

func TestCacheImmediateExpire(t *testing.T) {
	instantReleaseTrigger := func(ctx context.Context) <-chan struct{} {
		channel := make(chan struct{}, 1)
		channel <- struct{}{}
		return channel
	}

	cache := interfacename.NewInterfaceNameChache(
		context.TODO(),
		newDummyGenerator(),
		interfacename.WithReleaseTrigger(instantReleaseTrigger),
	)

	for _, nt := range CacheExpireTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
	}
}

func TestCacheExpire(t *testing.T) {
	instantReleaseTrigger := func(ctx context.Context) <-chan struct{} {
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
		newDummyGenerator(),
		interfacename.WithReleaseTrigger(instantReleaseTrigger),
	)

	for _, nt := range CacheExpireTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
		<-time.After(10 * time.Millisecond)
	}
}

func TestCacheReserveCached(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cache := interfacename.NewInterfaceNameChache(ctx, newDummyGenerator())
	for _, nt := range CacheGenerateTests {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}

	for _, nt := range CacheGenerateTests {
		name := cache.CheckAndReserve(nt.id, nt.expected, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}
}

func TestReserveUnused(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	instantReleaseTrigger := func(ctx context.Context) <-chan struct{} {
		channel := make(chan struct{}, 1)
		channel <- struct{}{}
		return channel
	}

	cache := interfacename.NewInterfaceNameChache(
		ctx,
		newDummyGenerator(),
		interfacename.WithReleaseTrigger(instantReleaseTrigger),
	)

	for _, nt := range CacheGenerateTests {
		name := cache.CheckAndReserve(nt.id, nt.expected, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}

	for _, nt := range CacheGenerateTests {
		name := cache.CheckAndReserve(nt.id, nt.expected, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}

	for _, nt := range CacheExpireTests {
		cache.Release(nt.id)
	}

	for _, nt := range CacheGenerateTests {
		name := cache.CheckAndReserve(nt.id, nt.expected, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}
}

func TestReserveError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	id1 := "first-id"
	id2 := "second-id"
	expected1 := fmt.Sprintf("%s-1", PREFIX)
	cache := interfacename.NewInterfaceNameChache(ctx, newDummyGenerator())

	name := cache.Generate(id1, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
	require.Equal(t, name, expected1)

	ret := cache.CheckAndReserve(id2, expected1, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
	require.Empty(t, ret)

	cache.Release(id1)
	<-time.After(10 * time.Microsecond)
	ret = cache.CheckAndReserve(id2, expected1, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
	require.Empty(t, ret)

	ret = cache.CheckAndReserve(id1, expected1, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
	require.Equal(t, ret, expected1)
}

func TestCacheCancelRelease(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(len(CacheCancelRelease))

	stuckReleaseTrigger := func(ctx context.Context) <-chan struct{} {
		channel := make(chan struct{}, 1)
		go func() {
			<-ctx.Done()
			wg.Done()
		}()
		return channel
	}

	cache := interfacename.NewInterfaceNameChache(
		ctx,
		newDummyGenerator(),
		interfacename.WithReleaseTrigger(stuckReleaseTrigger),
	)

	for _, nt := range CacheCancelRelease {
		name := cache.Generate(nt.id, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
		cache.Release(nt.id)
		//<-time.After(10 * time.Millisecond)
		name = cache.CheckAndReserve(nt.id, nt.expected, PREFIX, interfacename.MAX_INTERFACE_NAME_LENGTH)
		require.Equal(t, name, nt.expected)
	}
	wg.Wait()
}
