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

package watchhandler

import (
	"context"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/next"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/sirupsen/logrus"
)

type watchhandler struct {
	TargetRegistry types.TargetRegistry
	*next.NextTargetRegistryServerImpl
}

// NewServer provides an implementation of TargetRegistryServer with the
// support of the chaining feature. This implementation handles Watch
// calls by sending corresponding watched resources on every change.
func NewServer(targetRegistry types.TargetRegistry) *watchhandler {
	r := &watchhandler{
		TargetRegistry:               targetRegistry,
		NextTargetRegistryServerImpl: &next.NextTargetRegistryServerImpl{},
	}
	return r
}

func (wh *watchhandler) Watch(t *nspAPI.Target, watcher nspAPI.TargetRegistry_WatchServer) error {
	err := wh.NextTargetRegistryServerImpl.Watch(t, watcher)
	if err != nil {
		return err
	}
	targetWatcher, err := wh.TargetRegistry.Watch(context.TODO(), t)
	if err != nil {
		return err
	}
	wh.watcher(watcher, targetWatcher.ResultChan())
	targetWatcher.Stop()
	return nil
}

func (wh *watchhandler) watcher(watcher nspAPI.TargetRegistry_WatchServer, ch <-chan []*nspAPI.Target) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.TargetResponse{
				Targets: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}
