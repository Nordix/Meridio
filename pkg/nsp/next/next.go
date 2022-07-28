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

package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

// NextTargetRegistryServer is the interface representing TargetRegistryServer with the
// support of the chaining feature.
type NextTargetRegistryServer interface {
	nspAPI.TargetRegistryServer
	setNext(NextTargetRegistryServer)
}

type NextTargetRegistryServerImpl struct {
	nspAPI.UnimplementedTargetRegistryServer
	next NextTargetRegistryServer
}

// Register will call the Register function of the next chain element
// If the next element is nil, then &empty.Empty{}, nil will be returned.
func (ntrsi *NextTargetRegistryServerImpl) Register(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	if ntrsi.next == nil {
		return &empty.Empty{}, nil
	}
	return ntrsi.next.Register(ctx, target)
}

// Unregister will call the Unregister function of the next chain element
// If the next element is nil, then &empty.Empty{}, nil will be returned.
func (ntrsi *NextTargetRegistryServerImpl) Unregister(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	if ntrsi.next == nil {
		return &empty.Empty{}, nil
	}
	return ntrsi.next.Unregister(ctx, target)
}

// Watch will call the Watch function of the next chain element
// If the next element is nil, then nil will be returned.
func (ntrsi *NextTargetRegistryServerImpl) Watch(t *nspAPI.Target, watcher nspAPI.TargetRegistry_WatchServer) error {
	if ntrsi.next == nil {
		return nil
	}
	return ntrsi.next.Watch(t, watcher)
}

func (ntrsi *NextTargetRegistryServerImpl) setNext(ntrs NextTargetRegistryServer) {
	ntrsi.next = ntrs
}
