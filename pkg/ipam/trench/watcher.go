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

package trench

import (
	"context"
	"io"
	"time"

	"github.com/go-logr/logr"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
	"google.golang.org/grpc"
)

type ConduitWatcher struct {
	Ctx                        context.Context
	TrenchWatchers             []TrenchWatcher
	TrenchName                 string
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	logger                     logr.Logger
}

type TrenchWatcher interface {
	AddConduit(ctx context.Context, name string) (types.Conduit, error)
	RemoveConduit(ctx context.Context, name string) error
}

func NewConduitWatcher(ctx context.Context, nspConn *grpc.ClientConn, trenchName string, trenchWatchers []TrenchWatcher) (*ConduitWatcher, error) {
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(nspConn)

	cw := &ConduitWatcher{
		Ctx:                        ctx,
		TrenchName:                 trenchName,
		TrenchWatchers:             trenchWatchers,
		ConfigurationManagerClient: configurationManagerClient,
		logger:                     log.FromContextOrGlobal(ctx).WithValues("class", "ConduitWatcher"),
	}
	return cw, nil
}

func (cw *ConduitWatcher) Start() {
	err := retry.Do(func() error {
		toWatch := &nspAPI.Conduit{
			Trench: &nspAPI.Trench{
				Name: cw.TrenchName,
			},
		}
		watchConduitClient, err := cw.ConfigurationManagerClient.WatchConduit(cw.Ctx, toWatch)
		if err != nil {
			return err
		}
		for {
			conduitResponse, err := watchConduitClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			for _, conduit := range conduitResponse.Conduits { // todo: add and remove conduits by checking the existing ones
				for _, w := range cw.TrenchWatchers {
					_, err := w.AddConduit(cw.Ctx, conduit.GetName())
					if err != nil {
						cw.logger.Error(err, "AddConduit")
					}
				}
			}
		}
		return nil
	}, retry.WithContext(cw.Ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		log.Fatal(cw.logger, "ConduitWatcher", "error", err)
	}
}
