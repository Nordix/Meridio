/*
Copyright (c) 2022 Nordix Foundation
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

package main_test

import (
	"math"
	"testing"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	lb "github.com/nordix/meridio/cmd/stateless-lb"
	"github.com/stretchr/testify/assert"
)

func TestIdentifierOffsetGenerator(t *testing.T) {
	iog := lb.NewIdentifierOffsetGenerator(2000)

	stream := &nspAPI.Stream{
		Name: "stream-a",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 200,
	}
	generated, err := iog.Generate(stream)
	assert.Equal(t, iog.Start+0, generated)
	assert.Nil(t, err)

	stream = &nspAPI.Stream{
		Name: "stream-b",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 100,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+200, generated)
	assert.Nil(t, err)

	stream = &nspAPI.Stream{
		Name: "stream-a",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 200,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+0, generated)
	assert.Nil(t, err)

	stream = &nspAPI.Stream{
		Name: "stream-c",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 1,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+300, generated)
	assert.Nil(t, err)

	stream = &nspAPI.Stream{
		Name: "stream-d",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 1,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+301, generated)
	assert.Nil(t, err)

	iog.Release("stream-b")

	stream = &nspAPI.Stream{
		Name: "stream-e",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 50,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+200, generated)
	assert.Nil(t, err)

	stream = &nspAPI.Stream{
		Name: "stream-f",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 55,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+302, generated)
	assert.Nil(t, err)

	stream = &nspAPI.Stream{
		Name: "stream-g",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 50,
	}
	generated, err = iog.Generate(stream)
	assert.Equal(t, iog.Start+250, generated)
	assert.Nil(t, err)

}

func TestIdentifierOffsetGenerator_Err(t *testing.T) {
	iog := lb.NewIdentifierOffsetGenerator(math.MaxInt - 200)
	stream := &nspAPI.Stream{
		Name: "stream-a",
		Conduit: &nspAPI.Conduit{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		MaxTargets: 201,
	}
	_, err := iog.Generate(stream)
	assert.NotNil(t, err)
}
