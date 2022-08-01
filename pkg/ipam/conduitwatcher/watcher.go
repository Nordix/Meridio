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

package conduitwatcher

import (
	"context"
	"io"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/nordix/meridio/pkg/security/credentials"
	"google.golang.org/grpc"
)

type Trench interface {
	AddConduit(ctx context.Context, name string) (types.Conduit, error)
	RemoveConduit(ctx context.Context, name string) error
}

func Start(ctx context.Context, nspService string, trenchName string, trenches []Trench, logger log.Logger) error {
	configurationManagerClient, err := getConfigurationManagerClient(nspService)
	if err != nil {
		return err
	}
	loggerCtx := log.WithLogger(ctx, logger)
	return retry.Do(func() error {
		toWatch := &nspAPI.Conduit{
			Trench: &nspAPI.Trench{
				Name: trenchName,
			},
		}
		watchConduitClient, err := configurationManagerClient.WatchConduit(loggerCtx, toWatch)
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
				for _, trench := range trenches {
					_, err := trench.AddConduit(loggerCtx, conduit.GetName())
					if err != nil {
						logger.Warn("err AddConduit: %v", err)
					}
				}
			}
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
}

func getConfigurationManagerClient(nspService string) (nspAPI.ConfigurationManagerClient, error) {
	nspConn, err := grpc.Dial(nspService,
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil, err
	}
	return nspAPI.NewConfigurationManagerClient(nspConn), nil
}
