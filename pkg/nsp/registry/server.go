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

package registry

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/next"
	"github.com/nordix/meridio/pkg/nsp/types"
)

type registry struct {
	TargetRegistry types.TargetRegistry
	*next.NextTargetRegistryServerImpl
}

// NewServer provides an implementation of TargetRegistryServer with the
// support of the chaining feature. This implementation handles Register
// and Unregister calls by adding or removing data into a storage (e.g
// memory or sqlite)
func NewServer(targetRegistry types.TargetRegistry) *registry {
	r := &registry{
		TargetRegistry:               targetRegistry,
		NextTargetRegistryServerImpl: &next.NextTargetRegistryServerImpl{},
	}
	return r
}

func (r *registry) Register(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := r.TargetRegistry.Set(ctx, target)
	if err != nil {
		return &empty.Empty{}, err
	}
	return r.NextTargetRegistryServerImpl.Register(ctx, target)
}

func (r *registry) Unregister(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := r.TargetRegistry.Remove(ctx, target)
	if err != nil {
		return &empty.Empty{}, err
	}
	return r.NextTargetRegistryServerImpl.Unregister(ctx, target)
}
