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

	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/sirupsen/logrus"
)

const (
	disconnected = iota
	connected
)

type status int

type conduitConnect struct {
	conduit   types.Conduit
	status    status
	cancelCtx context.CancelFunc
	mu        sync.Mutex
	ctxMu     sync.Mutex
}

func newConduitConnect(conduit types.Conduit) *conduitConnect {
	cc := &conduitConnect{
		conduit: conduit,
		status:  disconnected,
	}
	return cc
}

func (cc *conduitConnect) connect() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if cc.status == connected {
		return
	}
	var ctx context.Context
	cc.ctxMu.Lock()
	ctx, cc.cancelCtx = context.WithCancel(context.TODO())
	cc.ctxMu.Unlock()
	for { // todo: retry
		if ctx.Err() != nil {
			return
		}
		err := cc.conduit.Connect(ctx)
		if err != nil {
			logrus.Warnf("error connecting conduit: %v ; %v", cc.conduit, err)
			continue
		}
		cc.status = connected
		break
	}
}

func (cc *conduitConnect) disconnect(ctx context.Context) error {
	cc.ctxMu.Lock()
	if cc.cancelCtx != nil {
		cc.cancelCtx()
	}
	cc.ctxMu.Unlock()
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.conduit.Disconnect(ctx)
}
