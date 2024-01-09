/*
Copyright (c) 2024 Nordix Foundation

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

package neighbor

import (
	"context"
	"sync"

	"github.com/vishvananda/netlink"
)

// NeighborMonitor -
// Receive neighbor update messages from kernel and propagate them to
// subscribers.
type NeighborMonitor struct {
	ch          chan netlink.NeighUpdate
	done        chan struct{}
	subscribers []NeighborMonitorSubscriber
	stateMask   int
	mu          sync.Mutex
}

// Subscribe -
func (nm *NeighborMonitor) Subscribe(subscriber NeighborMonitorSubscriber) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.subscribers = append(nm.subscribers, subscriber)
}

// UnSubscribe -
func (nm *NeighborMonitor) UnSubscribe(subscriber NeighborMonitorSubscriber) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	for index, current := range nm.subscribers {
		if subscriber == current {
			nm.subscribers = append(nm.subscribers[:index], nm.subscribers[index+1:]...)
		}
	}
}

// update -
// Passes neighbor events to subscribers.
func (nm *NeighborMonitor) update(neigh netlink.NeighUpdate) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	if nm.stateMask&neigh.State == 0 {
		return
	}
	for _, subscriber := range nm.subscribers {
		subscriber.NeighborUpdated(neigh)
	}
}

// start -
// Starts checking the event channel for updates.
// And checks if context is still open.
func (nm *NeighborMonitor) start(ctx context.Context) {
	for {
		select {
		case update, ok := <-nm.ch:
			if !ok {
				nm.Close() // doesn't make much sense, since done channel must have been closed
				return
			}
			nm.update(update)
		case <-ctx.Done():
			nm.Close()
			nm.mu.Lock()
			nm.subscribers = nm.subscribers[:0]
			nm.mu.Unlock()
			return
		}
	}
}

// Close -
func (nm *NeighborMonitor) Close() {
	close(nm.done)
}

// NewNeighborMonitor -
// Creates a new neighbor monitor. Monitor is kept open until either Close()
// is called or the context is closed.
func NewNeighborMonitor(ctx context.Context, options ...Option) (*NeighborMonitor, error) {
	opts := &neighOptions{
		stateMask: 0xffffffff,
	}
	for _, opt := range options {
		opt(opts)
	}

	neighborMonitor := &NeighborMonitor{
		ch:        make(chan netlink.NeighUpdate),
		done:      make(chan struct{}),
		stateMask: opts.stateMask,
	}

	err := netlink.NeighSubscribe(neighborMonitor.ch, neighborMonitor.done)
	if err != nil {
		return nil, err
	}
	go neighborMonitor.start(ctx)

	return neighborMonitor, nil
}

type Option func(o *neighOptions)

type neighOptions struct {
	stateMask int
}

// WithStateMask -
// WithStateMask returns a new Option with the given state.
func WithStateMask(stateMask int) Option {
	return func(o *neighOptions) {
		o.stateMask = stateMask
	}
}
