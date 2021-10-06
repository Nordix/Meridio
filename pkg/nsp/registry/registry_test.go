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

package registry_test

import (
	"testing"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry"
	"github.com/stretchr/testify/assert"
)

type getTest struct {
	parameter *nspAPI.Target
	result    []*nspAPI.Target
}

func Test_Set(t *testing.T) {
	targetRegistry := registry.New(nil)
	assert.NotNil(t, targetRegistry)
	assert.Equal(t, targetRegistry.Get(nil), []*nspAPI.Target{})
	for _, target := range getTargetList() {
		targetRegistry.Set(target)
	}
	assert.Equal(t, targetRegistry.Get(nil), getTargetList())
}

func Test_Remove(t *testing.T) {
	targetRegistry := registry.New(nil)
	assert.NotNil(t, targetRegistry)
	assert.Equal(t, []*nspAPI.Target{}, targetRegistry.Get(nil))
	for _, target := range getTargetList() {
		targetRegistry.Set(target)
	}
	for _, target := range getTargetList() {
		targetRegistry.Remove(target)
	}
	assert.Equal(t, []*nspAPI.Target{}, targetRegistry.Get(nil))
}

func Test_Get(t *testing.T) {
	getTests := []*getTest{
		{
			parameter: &nspAPI.Target{
				Status: nspAPI.Target_ENABLED,
				Type:   nspAPI.Target_DEFAULT,
				Stream: &nspAPI.Stream{
					Name: "stream-a",
					Conduit: &nspAPI.Conduit{
						Name: "conduit-a",
						Trench: &nspAPI.Trench{
							Name: "trench-a",
						},
					},
				},
			},
			result: []*nspAPI.Target{
				getTargetList()[1],
				getTargetList()[2],
			},
		},
		{
			parameter: &nspAPI.Target{
				Status: nspAPI.Target_ENABLED,
				Type:   nspAPI.Target_DEFAULT,
				Stream: &nspAPI.Stream{
					Conduit: &nspAPI.Conduit{
						Name: "conduit-a",
						Trench: &nspAPI.Trench{
							Name: "trench-a",
						},
					},
				},
			},
			result: []*nspAPI.Target{
				getTargetList()[1],
				getTargetList()[2],
			},
		},
		{
			parameter: &nspAPI.Target{
				Status: nspAPI.Target_ENABLED,
				Type:   nspAPI.Target_FRONTEND,
				Stream: nil,
			},
			result: []*nspAPI.Target{
				getTargetList()[5],
				getTargetList()[6],
			},
		},
		{
			parameter: &nspAPI.Target{
				Status: nspAPI.Target_ANY,
				Type:   nspAPI.Target_DEFAULT,
				Stream: &nspAPI.Stream{
					Conduit: &nspAPI.Conduit{
						Trench: &nspAPI.Trench{
							Name: "trench-a",
						},
					},
				},
			},
			result: []*nspAPI.Target{
				getTargetList()[0],
				getTargetList()[1],
				getTargetList()[2],
				getTargetList()[3],
			},
		},
	}
	targetRegistry := registry.New(nil)
	assert.NotNil(t, targetRegistry)
	assert.Equal(t, []*nspAPI.Target{}, targetRegistry.Get(nil))
	for _, target := range getTargetList() {
		targetRegistry.Set(target)
	}
	for _, gt := range getTests {
		targets := targetRegistry.Get(gt.parameter)
		assert.Equal(t, gt.result, targets)
	}
}

func getTargetList() []*nspAPI.Target {
	return []*nspAPI.Target{
		{
			Ips:    []string{"172.16.0.1/32", "fd00::1/32"},
			Status: nspAPI.Target_DISABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Ips:    []string{"172.16.0.2/32", "fd00::2/32"},
			Status: nspAPI.Target_ENABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Ips:    []string{"172.16.0.3/32", "fd00::3/32"},
			Status: nspAPI.Target_ENABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Ips:    []string{"172.16.0.3/32", "fd00::3/32"},
			Status: nspAPI.Target_ENABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-b",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Ips:    []string{"172.16.0.4/32", "fd00::4/32"},
			Status: nspAPI.Target_ENABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: nil,
		},
		{
			Ips:    []string{"211.10.0.1/32"},
			Status: nspAPI.Target_ENABLED,
			Type:   nspAPI.Target_FRONTEND,
			Stream: nil,
		},
		{
			Ips:    []string{"211.10.0.2/32"},
			Status: nspAPI.Target_ENABLED,
			Type:   nspAPI.Target_FRONTEND,
			Stream: nil,
		},
	}
}
