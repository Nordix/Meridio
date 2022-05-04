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

package keepalive

import (
	"context"
	"errors"
	"sync"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry/sqlite"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/sirupsen/logrus"
)

const (
	removeTimeout  = 5 * time.Second
	defaultTimeout = 60 * time.Second
)

type TimeoutTrigger func() <-chan struct{}

// KeepAlive is a middleware for the Target registry.
// It will monitor each target and check if they receive refresh
// notifications. If not, target will be removed.
type KeepAlive struct {
	TargetRegistry types.TargetRegistry
	Timeout        TimeoutTrigger
	mu             sync.Mutex
	targets        map[string]*keepAliveTarget
}

type keepAliveTarget struct {
	context       context.Context
	contextCancel context.CancelFunc
	target        *nspAPI.Target
}

// NewServer is the constructor of KeepAlive
func New(options ...Option) (*KeepAlive, error) {
	ka := &KeepAlive{
		TargetRegistry: nil,
		Timeout: func() <-chan struct{} {
			return delay(defaultTimeout)
		},
		targets: map[string]*keepAliveTarget{},
	}
	for _, opt := range options {
		opt(ka)
	}
	err := ka.restore()
	if err != nil {
		return nil, err
	}
	return ka, nil
}

// Set will add/update the target to the target registry and will wait for the target
// to be refreshed. If not refreshed on time, the target will be removed from the target
// registry.
// A target cannot be added with the enabled status, it has first to be disabled and then
// to be updated as enabled. This function will set the status to disabled if the target
// were not existing previously.
func (ka *KeepAlive) Set(ctx context.Context, target *nspAPI.Target) error {
	ka.mu.Lock()
	defer ka.mu.Unlock()
	_, exists := ka.targets[sqlite.GetTargetID(target)]
	// a target cannot register as enabled if it was not previously registered.
	if !exists {
		target.Status = nspAPI.Target_DISABLED
	}
	if ka.TargetRegistry != nil {
		// todo: cache to avoid setting multiple time the same target
		err := ka.TargetRegistry.Set(ctx, target)
		if err != nil {
			return err
		}
	}
	ka.add(target)
	return nil
}

// Remove will remove the target from the target registry and stop
// waiting for the target to be refreshed.
func (ka *KeepAlive) Remove(ctx context.Context, target *nspAPI.Target) error {
	ka.mu.Lock()
	defer ka.mu.Unlock()
	return ka.remove(ctx, target)
}

// Get returns the returned value by the target registry.
// if no set, an error is returned.
func (ka *KeepAlive) Watch(ctx context.Context, target *nspAPI.Target) (types.TargetWatcher, error) {
	if ka.TargetRegistry == nil {
		return nil, errors.New("the target registry is not set in the keepalive registry, and the keepalive registry doesn't support the watch function")
	}
	return ka.TargetRegistry.Watch(ctx, target)
}

// Get returns the returned value by the target registry.
// if no set, an empty list is returned.
func (ka *KeepAlive) Get(ctx context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
	if ka.TargetRegistry == nil {
		return []*nspAPI.Target{}, nil
	}
	return ka.TargetRegistry.Get(ctx, target)
}

func (ka *KeepAlive) add(target *nspAPI.Target) {
	kaTarget, exists := ka.targets[sqlite.GetTargetID(target)]
	if exists {
		logrus.Infof("Update/refresh: %v", target)
		kaTarget.contextCancel()
	} else {
		logrus.Infof("Register: %v", target)
	}
	ctx, cancel := context.WithCancel(context.TODO())
	ka.targets[sqlite.GetTargetID(target)] = &keepAliveTarget{
		context:       ctx,
		target:        target,
		contextCancel: cancel,
	}
	go func() {
		ka.timeout(ctx, target)
	}()
}

func (ka *KeepAlive) remove(ctx context.Context, target *nspAPI.Target) error {
	delete(ka.targets, sqlite.GetTargetID(target))
	logrus.Infof("Unregister: %v", target)
	if ka.TargetRegistry == nil {
		return nil
	}
	return ka.TargetRegistry.Remove(ctx, target)
}

func (ka *KeepAlive) timeout(ctx context.Context, target *nspAPI.Target) {
	select {
	case <-ka.Timeout():
		// no refresh
		ka.mu.Lock()
		defer ka.mu.Unlock()
		kaTarget, exists := ka.targets[sqlite.GetTargetID(target)]
		if !exists || ctx != kaTarget.context {
			return
		}
		removeCtx, cancel := context.WithTimeout(context.TODO(), removeTimeout)
		defer cancel()
		err := ka.remove(removeCtx, target)
		if err != nil {
			logrus.Errorf("error removing target after it expired: %v", err)
		}
	case <-ctx.Done():
		// cancel to refresh
		return
	}
}

// re-fill the target list with the ones in the registry
func (ka *KeepAlive) restore() error {
	defaultTargets, err := ka.Get(context.Background(), &nspAPI.Target{
		Status: nspAPI.Target_ANY,
		Type:   nspAPI.Target_DEFAULT,
	})
	if err != nil {
		return err
	}
	frontEndTargets, err := ka.Get(context.Background(), &nspAPI.Target{
		Status: nspAPI.Target_ANY,
		Type:   nspAPI.Target_FRONTEND,
	})
	if err != nil {
		return err
	}
	targets := append(defaultTargets, frontEndTargets...)
	for _, target := range targets {
		ka.add(target)
	}
	return nil
}
