/*
Copyright (c) 2024 Nordix Foundation
Copyright (c) 2025 OpenInfra Foundation Europe

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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	lbAPI "github.com/nordix/meridio/api/loadbalancer/v1"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/logutils"
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
//
// Upon Register/Unregister the update of watchers is performed asynchronously not
// blocking the caller. Watch fetches the most recent availability information to
// be sent to the client.
//
// Skips sending duplicate availability updates to a watcher, ensuring only changes
// are propagated.
type ForwardingAvailabilityService struct {
	lbAPI.UnimplementedStreamAvailabilityServiceServer
	ctx               context.Context
	forwardingPlane   map[string]struct{} // available forwarding paths
	target            *lbAPI.Target
	unavailableTarget *lbAPI.Target
	watchStream       map[chan<- chan struct{}]struct{} // update channels of watcher streams
	logger            logr.Logger
	mu                sync.RWMutex
	muWatcher         sync.Mutex
	targetRetriever   TargetGetter
}

func (fas *ForwardingAvailabilityService) Watch(_ *emptypb.Empty, watcher lbAPI.StreamAvailabilityService_WatchServer) error {
	// Update channel for Stream availability monitoring. Buffered to ensure no changes
	// go unnoticed. This allows non-blocking writes from Register/Unregister, avoiding
	// deadlocks.
	updateCh := make(chan chan struct{}, 1)
	fas.muWatcher.Lock()
	fas.watchStream[updateCh] = struct{}{}
	fas.muWatcher.Unlock()

	var lastSentTarget *lbAPI.Target // store last successfully sent target for send optimization

	// send initial availability response based on the actual state of the forwardingPlane
	target := fas.getTarget()
	fas.logger.V(1).Info("Watch", "initial target", target)
	err := watcher.Send(&lbAPI.Response{Targets: []*lbAPI.Target{target}})
	if err == nil {
		lastSentTarget = target
	} else {
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
				var err error
				target := fas.getTarget()
				fas.logger.V(1).Info("Watch", logutils.ToKV(
					logutils.LbApiTargetValue(target))...)
				if lastSentTarget == nil || !proto.Equal(lastSentTarget, target) {
					err = watcher.Send(&lbAPI.Response{Targets: []*lbAPI.Target{target}})
					if err == nil {
						lastSentTarget = target // Update lastSentTarget ONLY if send was successful
					}
				} else {
					fas.logger.V(1).Info("Watch skipped duplicate target update",
						logutils.ToKV(
							logutils.LbApiTargetValue(target))...)
					err = nil
				}
				close(feedbackCh) // Signal completion to the `update` caller regardless of send/skip
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
// Stop stops further user interaction via Register/Unregister calls. Blocks until all
// watchers have processed and acted upon the stop request. Also, informs clients that
// the loadbalancer target is no longer available.
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
				fas.logger.V(1).Info("Watcher notification sent")
				select {
				case <-feedbackCh:
					fas.logger.V(1).Info("Watcher processed notification")
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
	var wg sync.WaitGroup

	fas.muWatcher.Lock()
	wg.Add(len(fas.watchStream))

	for ch := range fas.watchStream {
		go func(ch chan<- chan struct{}) {
			defer wg.Done()
			select {
			case ch <- make(chan struct{}):
				fas.logger.V(1).Info("update event sent")
			case <-ctx.Done():
				return
			default:
				// The channel associated with watchStream must be a buffered one,
				// so that the watchers would not miss sending out the most recent
				// availability status.
				fas.logger.V(1).Info("update event skipped")
				return
			}
		}(ch)
	}

	fas.muWatcher.Unlock()
	wg.Wait()
}

func (fas *ForwardingAvailabilityService) getTarget() *lbAPI.Target {
	return fas.targetRetriever.Get(fas) // Call Get() on the injected interface
}

func (fas *ForwardingAvailabilityService) coreGetTarget() *lbAPI.Target {
	fas.mu.RLock()
	defer fas.mu.RUnlock()
	if len(fas.forwardingPlane) == 0 {
		return fas.unavailableTarget
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
		ctx:               ctx,
		logger:            logger,
		forwardingPlane:   make(map[string]struct{}),
		watchStream:       make(map[chan<- chan struct{}]struct{}),
		target:            target,
		unavailableTarget: &lbAPI.Target{},
		targetRetriever:   &defaultTargetGetter{},
	}

	return fas
}

// NewForwardingAvailabilityServiceForTest -
// Creates a new Test-specific forwarding availability service using a delayed TargetGetter.
func NewForwardingAvailabilityServiceForTest(
	ctx context.Context,
	target *lbAPI.Target,
	delay time.Duration,
) *ForwardingAvailabilityService {
	fas := NewForwardingAvailabilityService(ctx, target)
	// Decorate the default getter with a delay using the provided test context
	fas.targetRetriever = newDelayedTargetGetter(&defaultTargetGetter{}, delay, ctx)
	return fas
}

// TargetGetter defines the interface for retrieving the current load balancer target
type TargetGetter interface {
	// Get retrieves the current target based on the state of the provided ForwardingAvailabilityService.
	// It takes 'fas' as an argument because it needs to access fas's internal (locked) state.
	Get(fas *ForwardingAvailabilityService) *lbAPI.Target
}

// DefaultTargetGetter is the production implementation of the TargetGetter interface
type defaultTargetGetter struct{}

// Get implements the TargetGetter interface for production environments.
func (d *defaultTargetGetter) Get(fas *ForwardingAvailabilityService) *lbAPI.Target {
	return fas.coreGetTarget()
}

type delayedTargetGetter struct {
	inner  TargetGetter // The actual getter to delegate to
	delay  time.Duration
	ctx    context.Context // Context to make the delay cancellable
	logger logr.Logger
}

// Get implements the TargetGetter interface by first delaying, then delegating
func (d *delayedTargetGetter) Get(fas *ForwardingAvailabilityService) *lbAPI.Target {
	logger := d.logger.WithName("Get")
	// Perform the delay
	if d.delay != 0 {
		logger.V(1).Info("delaying call", "delay", d.delay)
		select {
		case <-d.ctx.Done():
			logger.V(1).Info("context done")
		case <-time.After(d.delay):
			logger.V(1).Info("delay expired")
		}
	}
	// Delegate to the inner (default) getter
	logger.V(1).Info("get target")
	return d.inner.Get(fas)
}

// newDelayedTargetGetter creates a new delayedTargetGetter
func newDelayedTargetGetter(inner TargetGetter, delay time.Duration, ctx context.Context) *delayedTargetGetter {
	return &delayedTargetGetter{
		inner:  inner,
		delay:  delay,
		ctx:    ctx,
		logger: log.FromContextOrGlobal(ctx).WithName("delayedTargetGetter"),
	}
}
