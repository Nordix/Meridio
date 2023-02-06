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

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry/sqlite"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestTargetRegistrySQLite_Set(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	dbFile := "test.db"
	os.Remove(dbFile)
	db, err := sqlite.New(dbFile)
	assert.Nil(t, err)
	defer func() {
		db.Close()
		os.Remove(dbFile)
	}()

	ctx := context.Background()

	nspTargets := []*nspAPI.Target{
		{
			Ips: []string{"172.16.0.1/32"},
			Context: map[string]string{
				"identifier": "32",
			},
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
			Ips: []string{"172.16.0.2/32"},
			Context: map[string]string{
				"identifier": "65",
			},
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
			Ips: []string{"172.16.0.3/32"},
			Context: map[string]string{
				"identifier": "1",
			},
			Status: nspAPI.Target_DISABLED,
			Type:   nspAPI.Target_DEFAULT,
			Stream: &nspAPI.Stream{
				Conduit: &nspAPI.Conduit{
					Trench: &nspAPI.Trench{},
				},
			},
		},
	}

	// Add nspTargets[0]
	err = db.Set(ctx, nspTargets[0])
	assert.Nil(t, err)

	targets, err := db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, nspTargets[0], targets[0])

	// Update nspTargets[0]
	nspTargets[0].Status = nspAPI.Target_ENABLED
	err = db.Set(ctx, nspTargets[0])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, nspTargets[0], targets[0])

	// Add nspTargets[1]
	err = db.Set(ctx, nspTargets[1])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 2)
	assert.Contains(t, targets, nspTargets[0])
	assert.Contains(t, targets, nspTargets[1])

	// Add nspTargets[2]
	err = db.Set(ctx, nspTargets[2])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 3)
	assert.Contains(t, targets, nspTargets[0])
	assert.Contains(t, targets, nspTargets[1])
	assert.Contains(t, targets, nspTargets[2])

	// Update nspTargets[0] with different conduit
	// Due the way the data are stored, 2 objects (Stream, Conduit, Trench) with the same name causes problems.
	// A new nspTargets[0] entry will be created and old nspTargets[0] and nspTargets[1] will be deleted due to stream name conflict.
	nspTargets[0].Stream.Conduit.Name = "conduit-b"
	nspTargets[0].Context["identifier"] = "100"
	err = db.Set(ctx, nspTargets[0])
	assert.Nil(t, err)

	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 2)
	assert.Contains(t, targets, nspTargets[0])
	assert.Contains(t, targets, nspTargets[2])

	// Remove nspTargets[0]
	err = db.Remove(ctx, nspTargets[0])
	assert.Nil(t, err)
	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 1)
	assert.Contains(t, targets, nspTargets[2])

	// Remove nspTargets[2]
	err = db.Remove(ctx, nspTargets[2])
	assert.Nil(t, err)
	targets, err = db.Get(ctx, &nspAPI.Target{Status: nspAPI.Target_ANY, Type: nspAPI.Target_DEFAULT})
	assert.Nil(t, err)
	assert.Len(t, targets, 0)
}
