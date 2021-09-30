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

package manager_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/configuration/manager"
	"github.com/nordix/meridio/pkg/configuration/manager/mocks"
	"github.com/nordix/meridio/pkg/configuration/registry"
	"github.com/stretchr/testify/assert"
)

type test struct {
	resource interface{}
	ch       interface{}
	err      bool
}

func Test_Register_Register(t *testing.T) {
	tests := []*test{
		{
			resource: nil,
			ch:       make(chan *nspAPI.Trench, 50),
			err:      false,
		},
		{
			resource: nil,
			ch:       nil,
			err:      true,
		},
		{
			resource: &nspAPI.Trench{
				Name: "trench-a",
			},
			ch:  make(chan *nspAPI.Trench, 50),
			err: false,
		},
		{
			resource: &nspAPI.Trench{
				Name: "trench-a",
			},
			ch:  make(chan *nspAPI.Conduit, 50),
			err: true,
		},
	}
	ctrl := gomock.NewController(t)
	configurationRegistry := mocks.NewMockConfigurationRegistry(ctrl)

	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	watcherNotifier := manager.NewWatcherNotifier(configurationRegistry, configurationEventChan)
	go watcherNotifier.Start(context.Background())

	configurationRegistry.EXPECT().GetTrench(gomock.Any()).Return(&nspAPI.Trench{})
	configurationRegistry.EXPECT().GetConduits(gomock.Any()).Return([]*nspAPI.Conduit{}).AnyTimes()
	configurationRegistry.EXPECT().GetStreams(gomock.Any()).Return([]*nspAPI.Stream{}).AnyTimes()
	configurationRegistry.EXPECT().GetFlows(gomock.Any()).Return([]*nspAPI.Flow{}).AnyTimes()
	configurationRegistry.EXPECT().GetVips(gomock.Any()).Return([]*nspAPI.Vip{}).AnyTimes()
	configurationRegistry.EXPECT().GetAttractors(gomock.Any()).Return([]*nspAPI.Attractor{}).AnyTimes()
	configurationRegistry.EXPECT().GetGateways(gomock.Any()).Return([]*nspAPI.Gateway{}).AnyTimes()

	for _, test := range tests {
		err := watcherNotifier.RegisterWatcher(test.resource, test.ch)
		if test.err {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func Test_Register_Unregister(t *testing.T) {
	trenchRequest := &nspAPI.Trench{
		Name: "trench-a",
	}

	ctrl := gomock.NewController(t)
	configurationRegistry := mocks.NewMockConfigurationRegistry(ctrl)
	configurationRegistry.EXPECT().GetTrench(gomock.Any()).DoAndReturn(func(trench *nspAPI.Trench) *nspAPI.Trench {
		assert.Equal(t, trenchRequest, trench)
		return trenchRequest
	}).AnyTimes()

	trenchChan := make(chan *nspAPI.Trench, 10)
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	watcherNotifier := manager.NewWatcherNotifier(configurationRegistry, configurationEventChan)
	go watcherNotifier.Start(context.Background())

	err := watcherNotifier.RegisterWatcher(trenchRequest, trenchChan)
	assert.Nil(t, err)

	var trenchResult *nspAPI.Trench
	select {
	case trenchResult = <-trenchChan:
	default:
	}
	assert.NotNil(t, trenchResult)
	assert.Equal(t, trenchResult, trenchRequest)

	watcherNotifier.UnregisterWatcher(trenchChan)
	// trench event
	trenchResult = nil
	configurationEventChan <- &registry.ConfigurationEvent{
		ResourceType: registry.Trench,
	}
	time.Sleep(500 * time.Millisecond)
	select {
	case trenchResult = <-trenchChan:
	default:
	}
	assert.Nil(t, trenchResult)
}

func Test_Trench_Event(t *testing.T) {
	trenchRequest := &nspAPI.Trench{
		Name: "trench-a",
	}

	ctrl := gomock.NewController(t)
	configurationRegistry := mocks.NewMockConfigurationRegistry(ctrl)
	configurationRegistry.EXPECT().GetTrench(gomock.Any()).DoAndReturn(func(trench *nspAPI.Trench) *nspAPI.Trench {
		assert.Equal(t, trenchRequest, trench)
		return trenchRequest
	}).AnyTimes()

	trenchChan := make(chan *nspAPI.Trench, 10)
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	watcherNotifier := manager.NewWatcherNotifier(configurationRegistry, configurationEventChan)
	go watcherNotifier.Start(context.Background())

	err := watcherNotifier.RegisterWatcher(trenchRequest, trenchChan)
	assert.Nil(t, err)
	var trenchResult *nspAPI.Trench
	select {
	case trenchResult = <-trenchChan:
	default:
	}
	assert.NotNil(t, trenchResult)
	assert.Equal(t, trenchResult, trenchRequest)

	// trench event
	trenchResult = nil
	configurationEventChan <- &registry.ConfigurationEvent{
		ResourceType: registry.Trench,
	}
	time.Sleep(500 * time.Millisecond)
	select {
	case trenchResult = <-trenchChan:
	default:
	}
	assert.NotNil(t, trenchResult)
	assert.Equal(t, trenchResult, trenchRequest)
}
