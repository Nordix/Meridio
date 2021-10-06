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

package nsp

import (
	"context"
	"errors"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/types"
)

type notifier struct {
	ch      chan<- []*nspAPI.Target
	toWatch *nspAPI.Target
}

type WatcherNotifier struct {
	TargetRegistry    types.TargetRegistry
	RegistryEventChan <-chan struct{}
	watchers          sync.Map // map[chan<- []*nspAPI.Target]notifier
}

func NewWatcherNotifier(targetRegistry types.TargetRegistry, registryEventChan <-chan struct{}) *WatcherNotifier {
	watcherNotifier := &WatcherNotifier{
		TargetRegistry:    targetRegistry,
		RegistryEventChan: registryEventChan,
	}
	return watcherNotifier
}

func (wn *WatcherNotifier) Start(context context.Context) {
	for { // todo: return if context is completed
		<-wn.RegistryEventChan
		wn.notifyWatchers()
	}
}

func (wn *WatcherNotifier) RegisterWatcher(toWatch *nspAPI.Target, ch chan<- []*nspAPI.Target) error {
	if ch == nil {
		return errors.New("channel cannot be nil")
	}
	notifier := &notifier{
		toWatch: toWatch,
		ch:      ch,
	}
	wn.watchers.Store(ch, notifier)
	notifier.notify(wn.TargetRegistry)
	return nil
}

func (wn *WatcherNotifier) UnregisterWatcher(ch chan<- []*nspAPI.Target) {
	wn.watchers.Delete(ch)
}

func (wn *WatcherNotifier) notifyWatchers() {
	wn.watchers.Range(func(_ interface{}, value interface{}) bool {
		notifier := value.(*notifier)
		notifier.notify(wn.TargetRegistry)
		return true
	})
}

func (n *notifier) notify(targetRegistry types.TargetRegistry) {
	targets := targetRegistry.Get(n.toWatch)
	// todo: check if same data has already been sent previously
	n.ch <- targets
}
