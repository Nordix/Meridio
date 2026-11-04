/*
Copyright (c) 2021-2022 Nordix Foundation

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

package keepalive_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry/common"
	keepAliveRegistry "github.com/nordix/meridio/pkg/nsp/registry/keepalive"
	"github.com/nordix/meridio/pkg/nsp/types/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func Test_Set(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	target := &nspAPI.Target{
		Ips:     []string{"192.168.1.1/32"},
		Status:  nspAPI.Target_ENABLED,
		Context: map[string]string{"identifier": "1"},
		Type:    nspAPI.Target_DEFAULT,
		Stream:  nil,
	}

	removeCtx, cancelRemove := context.WithCancel(context.TODO())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRegistry := mocks.NewMockTargetRegistry(ctrl)
	targetRegistry.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes() // constructor restore
	set := targetRegistry.EXPECT().Set(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, trgt *nspAPI.Target) error {
		assert.Equal(t, target.Ips, trgt.Ips)
		assert.Equal(t, target.Context, trgt.Context)
		assert.Equal(t, target.Type, trgt.Type)
		assert.Equal(t, target.Stream, trgt.Stream)
		assert.Equal(t, nspAPI.Target_DISABLED, trgt.Status) // check the first registry is set to disabled
		return nil
	})
	set2 := targetRegistry.EXPECT().Set(gomock.Any(), target).Return(nil).After(set)
	targetRegistry.EXPECT().Remove(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *nspAPI.Target) error {
		cancelRemove()
		return nil
	}).After(set2)

	timeoutCtx, cancelTimeout := context.WithCancel(context.TODO())
	keepAliveRegistry, err := keepAliveRegistry.New(keepAliveRegistry.WithRegistry(targetRegistry), keepAliveRegistry.WithContextTimeout(timeoutCtx))
	assert.Nil(t, err)

	err = keepAliveRegistry.Set(context.TODO(), target)
	assert.Nil(t, err)

	err = keepAliveRegistry.Set(context.TODO(), target)
	assert.Nil(t, err)

	cancelTimeout()
	<-removeCtx.Done()
}

func Test_Remove(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	target := &nspAPI.Target{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRegistry := mocks.NewMockTargetRegistry(ctrl)
	targetRegistry.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes() // constructor restore
	targetRegistry.EXPECT().Remove(gomock.Any(), target).Return(nil)

	keepAliveRegistry, err := keepAliveRegistry.New(keepAliveRegistry.WithRegistry(targetRegistry), keepAliveRegistry.WithTimeout(10*time.Microsecond))
	assert.Nil(t, err)

	err = keepAliveRegistry.Remove(context.TODO(), target)
	assert.Nil(t, err)
}

func Test_Watch(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	target := &nspAPI.Target{
		Status: nspAPI.Target_ANY,
	}
	watcher := common.NewRegistryWatcher(target)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRegistry := mocks.NewMockTargetRegistry(ctrl)
	targetRegistry.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes() // constructor restore
	targetRegistry.EXPECT().Watch(gomock.Any(), target).Return(watcher, nil)

	keepAliveRegistry, err := keepAliveRegistry.New(keepAliveRegistry.WithRegistry(targetRegistry), keepAliveRegistry.WithTimeout(10*time.Microsecond))
	assert.Nil(t, err)

	targetWatcher, err := keepAliveRegistry.Watch(context.TODO(), target)
	assert.Nil(t, err)
	assert.Equal(t, watcher, targetWatcher)
}

func Test_Get(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	target := &nspAPI.Target{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRegistry := mocks.NewMockTargetRegistry(ctrl)
	constructor := targetRegistry.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
	targetRegistry.EXPECT().Get(gomock.Any(), target).Return(nil, nil).After(constructor)

	keepAliveRegistry, err := keepAliveRegistry.New(keepAliveRegistry.WithRegistry(targetRegistry), keepAliveRegistry.WithTimeout(10*time.Microsecond))
	assert.Nil(t, err)

	targets, err := keepAliveRegistry.Get(context.TODO(), target)
	assert.Nil(t, err)
	assert.Empty(t, targets)
}

// Test_Set_AbandonedCall verifies that a Set whose context is already done
// returns its error without touching the registry or the keepalive entries.
// The mock registry has no Set expectation for that call, so a registry Set
// would fail the test.
func Test_Set_AbandonedCall(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	target := &nspAPI.Target{
		Ips:     []string{"192.168.1.1/32"},
		Status:  nspAPI.Target_ENABLED,
		Context: map[string]string{"identifier": "1"},
		Type:    nspAPI.Target_DEFAULT,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRegistry := mocks.NewMockTargetRegistry(ctrl)
	targetRegistry.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes() // constructor restore
	// The later, live Set is the first one that reaches the registry, so the
	// target is stored as disabled: the abandoned call left no keepalive entry.
	targetRegistry.EXPECT().Set(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, trgt *nspAPI.Target) error {
		assert.Equal(t, nspAPI.Target_DISABLED, trgt.Status)
		return nil
	})
	targetRegistry.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)

	keepAliveRegistry, err := keepAliveRegistry.New(keepAliveRegistry.WithRegistry(targetRegistry), keepAliveRegistry.WithTimeout(time.Hour))
	assert.Nil(t, err)

	// Step 1: the client already gave up.
	abandoned, cancel := context.WithCancel(context.TODO())
	cancel()
	err = keepAliveRegistry.Set(abandoned, target)
	assert.ErrorIs(t, err, context.Canceled)

	// Step 2: a live Set. If step 1 had added a keepalive entry, this would be
	// a refresh and keep the status ENABLED.
	assert.Nil(t, keepAliveRegistry.Set(context.TODO(), &nspAPI.Target{
		Ips: target.Ips, Status: nspAPI.Target_ENABLED, Context: target.Context, Type: target.Type,
	}))
	assert.Nil(t, keepAliveRegistry.Remove(context.TODO(), target))
}

// Test_Remove_StopsExpiry verifies that Unregister ends the entry's expiry
// goroutine and stops its timer instead of leaving them until the timeout.
// goleak at cleanup fails the test if the goroutine is still running.
func Test_Remove_StopsExpiry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	target := &nspAPI.Target{
		Ips:     []string{"192.168.1.1/32"},
		Context: map[string]string{"identifier": "1"},
		Type:    nspAPI.Target_DEFAULT,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	targetRegistry := mocks.NewMockTargetRegistry(ctrl)
	targetRegistry.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes() // constructor restore
	targetRegistry.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)
	targetRegistry.EXPECT().Remove(gomock.Any(), gomock.Any()).Return(nil)

	// A timeout far beyond the test: only Remove can end the expiry goroutine.
	keepAliveRegistry, err := keepAliveRegistry.New(keepAliveRegistry.WithRegistry(targetRegistry), keepAliveRegistry.WithTimeout(time.Hour))
	assert.Nil(t, err)

	assert.Nil(t, keepAliveRegistry.Set(context.TODO(), target))
	assert.Nil(t, keepAliveRegistry.Remove(context.TODO(), target))
}
