/*
Copyright (c) 2021-2023 Nordix Foundation

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

package trench

import (
	"context"
	"io"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
)

//go:generate mockgen -source=watcher.go -destination=mocks/watcher.go -package=mocks
type TrenchWatcher interface {
	AddConduit(ctx context.Context, name string) (types.Conduit, error)
	RemoveConduit(ctx context.Context, name string) error
}

func NewConduitWatcher(ctx context.Context, configurationManagerClient nspAPI.ConfigurationManagerClient, trenchName string, trenchWatchers []TrenchWatcher) {
	currentConduits := map[string]struct{}{}
	logger := log.FromContextOrGlobal(ctx).WithValues("class", "ConduitWatcher")

	_ = retry.Do(func() error {
		toWatch := &nspAPI.Conduit{
			Trench: &nspAPI.Trench{
				Name: trenchName,
			},
		}
		watchConduitClient, err := configurationManagerClient.WatchConduit(ctx, toWatch)
		if err != nil {
			logger.Error(err, "WatchConduit")
			return err
		}
		for {
			conduitResponse, err := watchConduitClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.Error(err, "watchConduitClient.Recv")
				return err
			}
			for _, w := range trenchWatchers {
				SetConduits(ctx, w, currentConduits, conduitResponse.GetConduits())
			}
			currentConduits = map[string]struct{}{}
			for _, conduit := range conduitResponse.GetConduits() {
				currentConduits[conduit.GetName()] = struct{}{}
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
}

// SetConduits adds or removes the conduit in the trenchWatcher based
// on the differences between currentConduits and newConduits
func SetConduits(ctx context.Context, tw TrenchWatcher, currentConduits map[string]struct{}, newConduits []*nspAPI.Conduit) {
	if tw == nil {
		return
	}
	cc := map[string]struct{}{}
	for k, v := range currentConduits {
		cc[k] = v
	}
	logger := log.FromContextOrGlobal(ctx).WithValues("class", "ConduitWatcher")
	// To add/update
	for _, conduit := range newConduits {
		// To add
		prefix, err := tw.AddConduit(ctx, conduit.GetName())
		if err != nil {
			logger.Error(err, "AddConduit", "Name", conduit.GetName())
		} else {
			logger.Info("AddConduit", "Name", conduit.GetName(), "prefix", prefix.GetCidr())
		}
		delete(cc, conduit.GetName())
	}
	// To remove
	for conduitName := range cc {
		err := tw.RemoveConduit(ctx, conduitName)
		if err != nil {
			logger.Error(err, "RemoveConduit", "Name", conduitName)
			continue
		}
		logger.Info("RemoveConduit", "Name", conduitName)
	}
}
