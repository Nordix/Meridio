/*
Copyright (c) 2021-2022 Nordix Foundation

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
	"sync"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
)

type conduitConnect struct {
	Timeout       time.Duration
	RetryDelay    time.Duration
	configuration *configurationImpl
	conduit       types.Conduit
	cancelOpen    context.CancelFunc
	ctxMu         sync.Mutex
}

func newConduitConnect(conduit types.Conduit, configurationManagerClient nspAPI.ConfigurationManagerClient) *conduitConnect {
	cc := &conduitConnect{
		conduit:       conduit,
		configuration: newConfigurationImpl(conduit.SetVIPs, conduit.GetConduit().ToNSP(), configurationManagerClient),
		Timeout:       10 * time.Second,
		RetryDelay:    2 * time.Second,
	}
	return cc
}

func (cc *conduitConnect) connect() {
	cc.ctxMu.Lock()
	if cc.cancelOpen != nil {
		return
	}
	ctx, cancelOpen := context.WithCancel(context.TODO())
	cc.cancelOpen = cancelOpen
	cc.ctxMu.Unlock()
	_ = retry.Do(func() error {
		retryCtx, cancel := context.WithTimeout(ctx, cc.Timeout) // todo: configurable timeout
		err := cc.conduit.Connect(retryCtx)
		defer cancel()
		if err != nil {
			log.Logger.Error(err, "connecting conduit", "conduit", cc.conduit.GetConduit())
			return err
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(cc.RetryDelay))

	cc.ctxMu.Lock()
	defer cc.ctxMu.Unlock()
	if ctx.Err() == nil {
		cc.configuration.Watch()
	}
}

func (cc *conduitConnect) disconnect(ctx context.Context) error {
	cc.ctxMu.Lock()
	defer cc.ctxMu.Unlock()
	if cc.cancelOpen != nil {
		cc.cancelOpen() // cancel open
	}
	cc.cancelOpen = nil
	cc.configuration.Stop()
	return cc.conduit.Disconnect(ctx)
}
