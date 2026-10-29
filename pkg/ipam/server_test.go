/*
Copyright (c) 2026 OpenInfra Foundation Europe

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

package ipam

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// missingConduitTrench is a trench whose conduits never exist; it counts lookups.
type missingConduitTrench struct {
	types.Prefix
	lookups atomic.Int32
}

func (t *missingConduitTrench) GetConduit(ctx context.Context, name string) (types.Conduit, error) {
	t.lookups.Add(1)
	return nil, nil
}

// Test_Allocate_WaitsBetweenConduitLookups verifies that an Allocate request
// whose conduit does not exist yet waits between lookups instead of polling
// the store back to back, and gives up when its ctx is done.
func Test_Allocate_WaitsBetweenConduitLookups(t *testing.T) {
	const trenchName = "trench-a"
	tr := &missingConduitTrench{
		Prefix: prefix.New(getTrenchName(trenchName, ipamAPI.IPFamily_IPV4), "172.16.0.0/16", nil),
	}
	is := &IpamServer{
		Trenches: map[ipamAPI.IPFamily]types.Trench{ipamAPI.IPFamily_IPV4: tr},
	}
	child := &ipamAPI.Child{
		Name: "child",
		Subnet: &ipamAPI.Subnet{
			Conduit: &nspAPI.Conduit{
				Name:   "conduit-a",
				Trench: &nspAPI.Trench{Name: trenchName},
			},
			Node:     "node-a",
			IpFamily: ipamAPI.IPFamily_IPV4,
		},
	}

	const deadline = 350 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	_, err := is.Allocate(ctx, child)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// About deadline/conduitPollInterval lookups are expected (4 with the
	// current values). Without the wait, the loop would issue a very large
	// number of lookups within the deadline.
	lookups := tr.lookups.Load()
	maxExpected := int32(deadline/conduitPollInterval) + 2
	assert.GreaterOrEqual(t, lookups, int32(2), "the lookup should be retried while waiting")
	assert.LessOrEqual(t, lookups, maxExpected, "lookups must be paced by conduitPollInterval")
}
