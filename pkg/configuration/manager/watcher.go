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

package manager

import (
	"context"
	"errors"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/configuration/registry"
)

type notifier interface {
	notify(ConfigurationRegistry)
}

type trenchNotifier struct {
	ch      chan<- *nspAPI.Trench
	toWatch *nspAPI.Trench
}

type conduitNotifier struct {
	ch      chan<- []*nspAPI.Conduit
	toWatch *nspAPI.Conduit
}

type streamNotifier struct {
	ch      chan<- []*nspAPI.Stream
	toWatch *nspAPI.Stream
}

type flowNotifier struct {
	ch      chan<- []*nspAPI.Flow
	toWatch *nspAPI.Flow
}

type vipNotifier struct {
	ch      chan<- []*nspAPI.Vip
	toWatch *nspAPI.Vip
}

type attractorNotifier struct {
	ch      chan<- []*nspAPI.Attractor
	toWatch *nspAPI.Attractor
}

type gatewayNotifier struct {
	ch      chan<- []*nspAPI.Gateway
	toWatch *nspAPI.Gateway
}

// WatcherNotifier implements WatcherRegistry
// On any event from configurationEventChan, WatcherNotifier will
// send an event with the corresponding resources to every concerned
// watchers using the get functions from ConfigurationRegistry
type WatcherNotifier struct {
	ConfigurationRegistry  ConfigurationRegistry
	configurationEventChan <-chan *registry.ConfigurationEvent
	watchers               map[registry.ResourceType]*sync.Map // map[<-chan *registry.ConfigurationEvent]notifier
}

// NewServer is the constructor of WatcherNotifier
func NewWatcherNotifier(configurationRegistry ConfigurationRegistry, configurationEventChan <-chan *registry.ConfigurationEvent) *WatcherNotifier {
	watcherNotifier := &WatcherNotifier{
		ConfigurationRegistry:  configurationRegistry,
		configurationEventChan: configurationEventChan,
		watchers:               make(map[registry.ResourceType]*sync.Map),
	}
	watcherNotifier.watchers[registry.Trench] = &sync.Map{}
	watcherNotifier.watchers[registry.Conduit] = &sync.Map{}
	watcherNotifier.watchers[registry.Stream] = &sync.Map{}
	watcherNotifier.watchers[registry.Flow] = &sync.Map{}
	watcherNotifier.watchers[registry.Vip] = &sync.Map{}
	watcherNotifier.watchers[registry.Attractor] = &sync.Map{}
	watcherNotifier.watchers[registry.Gateway] = &sync.Map{}
	return watcherNotifier
}

// Start is receiving event fron the configurationEventChan channel
// and notifier the corresponding subscribers about the changes
func (wn *WatcherNotifier) Start(context context.Context) {
	for { // todo: return if context is completed
		event := <-wn.configurationEventChan
		wn.notifyWatchers(event.ResourceType)
	}
}

// RegisterWatcher registers a watcher
// toWatch is the resource to watch, it can be nil or should be a pointer to a
// resource struct from the nsp API.
// ch is a channel where the events will be send on any change in the configuration.
// ch should receive a single resource for the trench, and a list for the other
// resources.
// if toWatch, ch should receive the same type of resource as toWatch.
// list of resources: Trench, Conduit, Stream, Flow, Vip, Attractor, Gateway.
func (wn *WatcherNotifier) RegisterWatcher(toWatch interface{}, ch interface{}) error {
	if ch == nil {
		return errors.New("channel cannot be nil")
	}
	resourceType := wn.checkResource(toWatch)
	ChanType := wn.checkChannel(ch)
	watchers, exists := wn.watchers[ChanType]
	if !exists {
		return errors.New("type does not exist")
	}
	if toWatch != nil && resourceType != ChanType {
		return errors.New("wrong type registered")
	}
	switch resourceType {
	case registry.Trench:
		trenchNotifier := &trenchNotifier{
			toWatch: toWatch.(*nspAPI.Trench),
			ch:      ch.(chan *nspAPI.Trench),
		}
		watchers.Store(ch, trenchNotifier)
		trenchNotifier.notify(wn.ConfigurationRegistry)
	case registry.Conduit:
		conduitNotifier := &conduitNotifier{
			toWatch: toWatch.(*nspAPI.Conduit),
			ch:      ch.(chan []*nspAPI.Conduit),
		}
		watchers.Store(ch, conduitNotifier)
		conduitNotifier.notify(wn.ConfigurationRegistry)
	case registry.Stream:
		streamNotifier := &streamNotifier{
			toWatch: toWatch.(*nspAPI.Stream),
			ch:      ch.(chan []*nspAPI.Stream),
		}
		watchers.Store(ch, streamNotifier)
		streamNotifier.notify(wn.ConfigurationRegistry)
	case registry.Flow:
		flowNotifier := &flowNotifier{
			toWatch: toWatch.(*nspAPI.Flow),
			ch:      ch.(chan []*nspAPI.Flow),
		}
		watchers.Store(ch, flowNotifier)
		flowNotifier.notify(wn.ConfigurationRegistry)
	case registry.Vip:
		vipNotifier := &vipNotifier{
			toWatch: toWatch.(*nspAPI.Vip),
			ch:      ch.(chan []*nspAPI.Vip),
		}
		watchers.Store(ch, vipNotifier)
		vipNotifier.notify(wn.ConfigurationRegistry)
	case registry.Attractor:
		attractorNotifier := &attractorNotifier{
			toWatch: toWatch.(*nspAPI.Attractor),
			ch:      ch.(chan []*nspAPI.Attractor),
		}
		watchers.Store(ch, attractorNotifier)
		attractorNotifier.notify(wn.ConfigurationRegistry)
	case registry.Gateway:
		gatewayNotifier := &gatewayNotifier{
			toWatch: toWatch.(*nspAPI.Gateway),
			ch:      ch.(chan []*nspAPI.Gateway),
		}
		watchers.Store(ch, gatewayNotifier)
		gatewayNotifier.notify(wn.ConfigurationRegistry)
	default:
	}
	return nil
}

// UnregisterWatcher unregisters a watcher
func (wn *WatcherNotifier) UnregisterWatcher(ch interface{}) {
	ChanType := wn.checkChannel(ch)
	watchers, exists := wn.watchers[ChanType]
	if !exists {
		return
	}
	watchers.Delete(ch)
}

func (wn *WatcherNotifier) notifyWatchers(resourceType registry.ResourceType) {
	watchers, exists := wn.watchers[resourceType]
	if !exists {
		return
	}
	watchers.Range(func(_ interface{}, value interface{}) bool {
		notifier := value.(notifier)
		notifier.notify(wn.ConfigurationRegistry)
		return true
	})
}

func (tn *trenchNotifier) notify(configurationRegistry ConfigurationRegistry) {
	trench := configurationRegistry.GetTrench(tn.toWatch)
	// todo: check if same data has already been sent previously
	tn.ch <- trench
}

func (cn *conduitNotifier) notify(configurationRegistry ConfigurationRegistry) {
	conduits := configurationRegistry.GetConduits(cn.toWatch)
	// todo: check if same data has already been sent previously
	cn.ch <- conduits
}

func (sn *streamNotifier) notify(configurationRegistry ConfigurationRegistry) {
	streams := configurationRegistry.GetStreams(sn.toWatch)
	// todo: check if same data has already been sent previously
	sn.ch <- streams
}

func (cn *flowNotifier) notify(configurationRegistry ConfigurationRegistry) {
	flows := configurationRegistry.GetFlows(cn.toWatch)
	// todo: check if same data has already been sent previously
	cn.ch <- flows
}

func (cn *vipNotifier) notify(configurationRegistry ConfigurationRegistry) {
	vips := configurationRegistry.GetVips(cn.toWatch)
	// todo: check if same data has already been sent previously
	cn.ch <- vips
}

func (cn *attractorNotifier) notify(configurationRegistry ConfigurationRegistry) {
	attractors := configurationRegistry.GetAttractors(cn.toWatch)
	// todo: check if same data has already been sent previously
	cn.ch <- attractors
}

func (cn *gatewayNotifier) notify(configurationRegistry ConfigurationRegistry) {
	gateways := configurationRegistry.GetGateways(cn.toWatch)
	// todo: check if same data has already been sent previously
	cn.ch <- gateways
}

func (wn *WatcherNotifier) checkResource(resource interface{}) registry.ResourceType {
	switch resource.(type) {
	case *nspAPI.Trench:
		return registry.Trench
	case *nspAPI.Conduit:
		return registry.Conduit
	case *nspAPI.Stream:
		return registry.Stream
	case *nspAPI.Flow:
		return registry.Flow
	case *nspAPI.Vip:
		return registry.Vip
	case *nspAPI.Attractor:
		return registry.Attractor
	case *nspAPI.Gateway:
		return registry.Gateway
	default:
	}
	return -1
}

func (wn *WatcherNotifier) checkChannel(ch interface{}) registry.ResourceType {
	switch ch.(type) {
	case chan *nspAPI.Trench:
		return registry.Trench
	case chan []*nspAPI.Conduit:
		return registry.Conduit
	case chan []*nspAPI.Stream:
		return registry.Stream
	case chan []*nspAPI.Flow:
		return registry.Flow
	case chan []*nspAPI.Vip:
		return registry.Vip
	case chan []*nspAPI.Attractor:
		return registry.Attractor
	case chan []*nspAPI.Gateway:
		return registry.Gateway
	default:
	}
	return -1
}
