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

package stream

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/protobuf/types/known/emptypb"

	lbAPI "github.com/nordix/meridio/api/loadbalancer/v1"
	"github.com/nordix/meridio/pkg/log"
)

// ForwardingAvailabilityService keeps track of forwarding plane availability
// towards application targets.
//
// It aggregates path informatation received form invidual Streams, and updates
// Watch clients based on the accumulated information whether the loadbalancer
// is capable of forwarding incoming traffic.
//
// Update is performed using a custom loadbalancer Target. There is one common
// loadbalancer Target for all local forwarding paths. (That is, the forwarding
// capability is not distinguished on Stream level currently.)
type ForwardingAvailabilityService struct {
	lbAPI.UnimplementedStreamAvailabilityServiceServer
	ctx             context.Context
	forwardingPlane map[string]struct{} // available forwarding paths
	target          *lbAPI.Target
	watchStream     map[chan<- chan struct{}]struct{} // update channels of watcher streams
	logger          logr.Logger
	mu              sync.RWMutex
	muWatcher       sync.Mutex
}

func (fas *ForwardingAvailabilityService) Watch(_ *emptypb.Empty, watcher lbAPI.StreamAvailabilityService_WatchServer) error {
	// add update channel to watch Stream availability
	updateCh := make(chan chan struct{})
	fas.muWatcher.Lock()
	fas.watchStream[updateCh] = struct{}{}
	fas.muWatcher.Unlock()

	// send initial availability response based on the actual state of the forwardingPlane
	target := fas.getTarget()
	fas.logger.V(1).Info("Watch", "initial target", target)
	err := watcher.Send(&lbAPI.Response{Targets: []*lbAPI.Target{target}})
	if err != nil {
		fas.logger.Error(err, "sending initial availability response")
		fas.muWatcher.Lock()
		delete(fas.watchStream, updateCh)
		fas.muWatcher.Unlock()
		return err
	}

	// listen for updates
	for {
		select {
		case <-watcher.Context().Done():
			fas.logger.V(1).Info("closing watcher")
			fas.muWatcher.Lock()
			delete(fas.watchStream, updateCh)
			fas.muWatcher.Unlock()
			return nil
		case feedbackCh, ok := <-updateCh: // an event on updateCh is an indication of update
			if ok {
				target := fas.getTarget()
				fas.logger.V(1).Info("Watch", "target", target)
				err := watcher.Send(&lbAPI.Response{Targets: []*lbAPI.Target{target}})
				close(feedbackCh) // indicate Send() has returned if anyone is interested
				if err != nil {
					fas.logger.Error(err, "sending availability response")
					fas.muWatcher.Lock()
					delete(fas.watchStream, updateCh)
					fas.muWatcher.Unlock()
					return err
				}
			}
		}
	}
}

// Register -
// Registers a new forwarding path. In case of the first forwarding
// path update() is called to inform Watch clients the LB is ready
// to forward traffic.
func (fas *ForwardingAvailabilityService) Register(name string) {
	fas.mu.Lock()
	defer fas.mu.Unlock()
	if fas.forwardingPlane == nil {
		return
	}
	if _, ok := fas.forwardingPlane[name]; ok { // already registered
		return
	}

	fas.logger.V(1).Info("Register stream forwarding path", "name", name)
	if len(fas.forwardingPlane) == 0 {
		fas.update(fas.ctx)
	}
	fas.forwardingPlane[name] = struct{}{}
}

// Unregister -
// Unregisters a forfarding path. In case there are no remaining forwarding
// paths, update informs the Watch clients.
func (fas *ForwardingAvailabilityService) Unregister(name string) {
	fas.mu.Lock()
	defer fas.mu.Unlock()
	if fas.forwardingPlane == nil {
		return
	}
	if _, ok := fas.forwardingPlane[name]; ok {
		fas.logger.V(1).Info("Unregister stream forwarding path", "name", name)
		delete(fas.forwardingPlane, name)
	}
	if len(fas.forwardingPlane) == 0 {
		fas.update(fas.ctx)
	}
}

// Stop -
// Stops further user interaction via Register/Unregister calls.
// Also informs clients that the loadbalancer target is no longer available.
func (fas *ForwardingAvailabilityService) Stop() {
	fas.mu.Lock()
	if fas.forwardingPlane == nil {
		fas.mu.Unlock()
		return
	}

	fas.logger.Info("Stop forwarding availability service")
	var wg sync.WaitGroup
	fas.forwardingPlane = nil
	// 2 seconds timeout to inform watcher clients
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fas.muWatcher.Lock()
	wg.Add(len(fas.watchStream))
	for ch := range fas.watchStream {
		go func(ch chan<- chan struct{}) {
			feedbackCh := make(chan struct{})
			defer wg.Done()
			select {
			case <-ctx.Done():
			case ch <- feedbackCh:
				fas.logger.V(1).Info("Watcher notified")
				select {
				case <-feedbackCh:
					fas.logger.V(1).Info("Watcher returned")
				case <-ctx.Done():
				}
			}
		}(ch)
	}
	fas.muWatcher.Unlock()
	fas.mu.Unlock()
	wg.Wait() // locks must be released first so that watchers are not blocked
}

// Note: Normally there should be only a single watcher (collocated Frontend)
// at a time, so not much point to identify watchers.
func (fas *ForwardingAvailabilityService) update(ctx context.Context) {
	fas.logger.V(1).Info("update")
	fas.muWatcher.Lock()
	defer fas.muWatcher.Unlock()
	for ch := range fas.watchStream {
		select {
		case ch <- make(chan struct{}):
			fas.logger.V(1).Info("watcher notified")
		case <-ctx.Done():
			return
		}
	}
}

func (fas *ForwardingAvailabilityService) getTarget() *lbAPI.Target {
	fas.mu.Lock()
	defer fas.mu.Unlock()
	if len(fas.forwardingPlane) == 0 {
		return &lbAPI.Target{}
	}
	return fas.target
}

// NewForwardingAvailabilityService -
// Creates a new forwarding availability service.
func NewForwardingAvailabilityService(ctx context.Context,
	target *lbAPI.Target) *ForwardingAvailabilityService {

	logger := log.Logger.WithValues("class", "ForwardingAvailabilityService")
	logger.Info("Creating forwarding availability service")
	fas := &ForwardingAvailabilityService{
		ctx:             ctx,
		logger:          logger,
		forwardingPlane: make(map[string]struct{}),
		watchStream:     make(map[chan<- chan struct{}]struct{}),
		target:          target,
	}

	return fas
}
