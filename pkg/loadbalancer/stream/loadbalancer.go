/*
Copyright (c) 2021-2024 Nordix Foundation

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
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/kernel/neighbor"
	"github.com/nordix/meridio/pkg/loadbalancer/flow"
	targetMetrics "github.com/nordix/meridio/pkg/loadbalancer/target"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/nordix/meridio/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

var errNoTarget error = errors.New("the target is not existing")

// LoadBalancer -
type LoadBalancer struct {
	*nspAPI.Stream
	TargetRegistryClient       nspAPI.TargetRegistryClient
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	IdentifierOffset           int
	nfqlb                      types.NFQueueLoadBalancer
	flows                      map[string]types.Flow
	targets                    map[int]types.Target // key: Identifier
	netUtils                   networking.Utils
	nfqueue                    int
	mu                         sync.Mutex
	ctx                        context.Context
	cancel                     context.CancelFunc
	pendingTargets             map[int]types.Target // key: Identifier
	defrag                     *Defrag
	pendingCh                  chan struct{} // trigger pending Targets processing
	logger                     logr.Logger
	targetHitsMetrics          *targetMetrics.HitsMetrics
	neighborReachDetector      *neighbor.NeighborReachabilityDetector
}

func New(
	stream *nspAPI.Stream,
	targetRegistryClient nspAPI.TargetRegistryClient,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	nfqueue int,
	netUtils networking.Utils,
	lbFactory types.NFQueueLoadBalancerFactory,
	identifierOffset int,
	targetHitsMetrics *targetMetrics.HitsMetrics,
	neighborReachDetector *neighbor.NeighborReachabilityDetector,
) (types.Stream, error) {
	n := int(stream.GetMaxTargets())
	m := int(stream.GetMaxTargets()) * 100
	nfqlb, err := lbFactory.New(stream.GetName(), m, n)
	if err != nil {
		return nil, fmt.Errorf("failed to create new nfqlb instance (%s): %w", stream.GetName(), err)
	}

	logger := log.Logger.WithValues("class", "LoadBalancer",
		"instance", stream.GetName(), "identifierOffset", identifierOffset)
	loadBalancer := &LoadBalancer{
		Stream:                     stream,
		TargetRegistryClient:       targetRegistryClient,
		ConfigurationManagerClient: configurationManagerClient,
		IdentifierOffset:           identifierOffset,
		flows:                      make(map[string]types.Flow),
		nfqlb:                      nfqlb,
		targets:                    make(map[int]types.Target),
		netUtils:                   netUtils,
		nfqueue:                    nfqueue,
		pendingTargets:             make(map[int]types.Target),
		pendingCh:                  make(chan struct{}, 10),
		logger:                     logger,
		targetHitsMetrics:          targetHitsMetrics,
		neighborReachDetector:      neighborReachDetector,
	}
	// first enable kernel's IP defrag except for the interfaces facing targets
	// (defrag is needed by Flows to match rules with L4 information)
	//loadBalancer.defrag, err = NewDefrag(types.InterfaceNamePrefix)
	loadBalancer.defrag, err = NewDefrag(GetInterfaceNamePrefix())
	if err != nil {
		logger.Error(err, "LB instance defrag setup")
		return nil, fmt.Errorf("failed to setup defrag: %w", err)
	}
	err = nfqlb.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start nfqlb instance (%s): %w", stream.GetName(), err)
	}
	logger.V(1).Info("Created LB instance")
	return loadBalancer, nil
}

func (lb *LoadBalancer) Start(ctx context.Context) error {
	lb.ctx, lb.cancel = context.WithCancel(ctx)
	if interfaceMonitor := lb.netUtils.GetInterfaceMonitor(lb.ctx); interfaceMonitor != nil {
		// register receiving interface events to trigger processing of pending Targets
		// whenever new NSM interfaces show up (to address race between NSP and NSM)
		interfaceMonitor.Subscribe(lb)
	}
	go func() { // todo
		lb.logger.V(1).Info("Watch LB Targets")
		err := retry.Do(func() error {
			return lb.watchTargets(lb.ctx)
		}, retry.WithContext(lb.ctx),
			retry.WithDelay(500*time.Millisecond),
			retry.WithErrorIngnored())
		if err != nil && lb.ctx.Err() != context.Canceled {
			lb.logger.Error(err, "watchTargets")
		}
	}()
	go lb.processPendingTargets(lb.ctx)
	lb.logger.V(1).Info("Watch LB Flows")
	err := retry.Do(func() error {
		return lb.watchFlows(lb.ctx)
	}, retry.WithContext(lb.ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		return fmt.Errorf("lb (%s) failed watching flows: %w", lb.Stream.GetName(), err)
	}
	return nil
}

func (lb *LoadBalancer) Delete() error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if lb.cancel != nil {
		lb.cancel()
	}
	var errFinal error
	for identifier := range lb.targets {
		err := lb.RemoveTarget(identifier)
		if err != nil {
			errFinal = utils.AppendErr(errFinal, fmt.Errorf("target: %w", err))
		}
	}
	for _, flow := range lb.flows {
		err := flow.Delete()
		if err != nil {
			errFinal = utils.AppendErr(errFinal, fmt.Errorf("flow: %w", err))
		}
	}
	err := lb.nfqlb.Delete()
	if err != nil {
		errFinal = utils.AppendErr(errFinal, fmt.Errorf("nfqlb: %w", err))
	}
	if interfaceMonitor := lb.netUtils.GetInterfaceMonitor(lb.ctx); interfaceMonitor != nil {
		// unregister receiving interface events
		interfaceMonitor.UnSubscribe(lb)
	}
	lb.logger.Info("Deleted")
	return errFinal // todo
}

// AddTarget -
// Adds a target by configuring routing and activating it in nfqlb.
// Note: configuration fails if no (NSM) inteface is available in the subnet
// the target IPs belong to.
func (lb *LoadBalancer) AddTarget(target types.Target) error {
	exists := lb.targetExists(target.GetIdentifier())
	if exists {
		return errors.New("the target is already registered")
	}
	logger := lb.logger.WithValues("func", "AddTarget", "target", target)
	err := target.Configure() // TODO: avoid multiple identical ip rule entries (e.g. after container crash)
	if err != nil {
		lb.addPendingTarget(target)
		returnErr := fmt.Errorf("target configure error: %w", err)
		err = target.Delete() // remove setup for any Target IP successfully configured
		if err != nil {
			return fmt.Errorf("%w; target delete error: %w", returnErr, err)
		}
		return returnErr
	}
	err = lb.nfqlb.Activate(target.GetIdentifier(), target.GetIdentifier()+lb.IdentifierOffset)
	if err != nil {
		lb.addPendingTarget(target)
		returnErr := fmt.Errorf("target activate error: %w", err)
		err = target.Delete()
		if err != nil {
			return fmt.Errorf("%w; target delete error: %w", returnErr, err)
		}
		return returnErr
	}
	lb.targets[target.GetIdentifier()] = target
	lb.neighborReachDetector.Register(target.GetIps()...)
	lb.removeFromPendingTarget(target)
	logger.Info("Added target")
	return nil
}

// RemoveTarget -
func (lb *LoadBalancer) RemoveTarget(identifier int) error {
	lb.removeFromPendingTargetByIdentifier(identifier)
	target := lb.getTarget(identifier)
	if target == nil {
		return errNoTarget
	}
	lb.neighborReachDetector.Unregister(target.GetIps()...)
	logger := lb.logger.WithValues("func", "RemoveTarget", "target", target)
	var errFinal error
	err := lb.nfqlb.Deactivate(target.GetIdentifier())
	if err != nil {
		errFinal = utils.AppendErr(errFinal, fmt.Errorf("target deactivate error: %w", err)) // todo
	}
	err = target.Delete()
	if err != nil {
		errFinal = utils.AppendErr(errFinal, fmt.Errorf("target delete error: %w", err)) // todo
	}
	delete(lb.targets, target.GetIdentifier())
	logger.Info("Removed target", "target", target)
	return errFinal
}

// TargetExists -
func (lb *LoadBalancer) TargetExists(identifier int) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.targetExists(identifier)
}

func (lb *LoadBalancer) GetTargets() []types.Target {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	targets := []types.Target{}
	for _, target := range lb.targets {
		targets = append(targets, target)
	}
	return targets
}

func (lb *LoadBalancer) targetExists(identifier int) bool {
	return lb.getTarget(identifier) != nil
}

func (lb *LoadBalancer) getTarget(identifier int) types.Target {
	return lb.targets[identifier]
}

// todo
func (lb *LoadBalancer) watchFlows(ctx context.Context) error {
	flowsToWatch := &nspAPI.Flow{
		Stream: lb.Stream,
	}
	watchFlow, err := lb.ConfigurationManagerClient.WatchFlow(ctx, flowsToWatch)
	if err != nil {
		return fmt.Errorf("failed to create configuration manager flow watcher (%s): %w",
			lb.Stream.Name, err)
	}
	for {
		flowResponse, err := watchFlow.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("configuration manager flow watcher receive error (%s): %w",
				lb.Stream.Name, err)
		}
		lb.setFlows(flowResponse.Flows)
	}
	return nil
}

func (lb *LoadBalancer) setFlows(flows []*nspAPI.Flow) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if !lb.isStreamRunning() {
		return
	}
	remainingFlows := make(map[string]struct{})
	for name := range lb.flows {
		remainingFlows[name] = struct{}{}
	}
	for _, f := range flows {
		fl, exists := lb.flows[f.GetName()]
		if !exists { // create
			newFlow, err := flow.New(f, lb.nfqlb)
			if err != nil {
				lb.logger.Error(err, "New flow")
				continue
			}
			lb.flows[f.GetName()] = newFlow
		} else { // update
			err := fl.Update(f)
			if err != nil {
				lb.logger.Error(err, "Flow Update")
				continue
			}
		}
		delete(remainingFlows, f.GetName())
	}
	// delete remaining flows
	for name := range remainingFlows {
		flow, exists := lb.flows[name]
		if !exists {
			continue
		}
		err := flow.Delete()
		if err != nil {
			lb.logger.Error(err, "Flow Delete")
		}
		delete(lb.flows, name)
	}
	// check if flow service can be enabled (needs at least 1 flow)
	// TODO: no flows in any of the streams?
	if len(lb.flows) > 0 {
		health.SetServingStatus(lb.ctx, health.FlowSvc, true)
	}
}

// todo
// watchTargets -
// Watches the configuration (NSP) to learn the available Targets to configure.
func (lb *LoadBalancer) watchTargets(ctx context.Context) error {
	watchTarget, err := lb.TargetRegistryClient.Watch(ctx, &nspAPI.Target{
		Status: nspAPI.Target_ENABLED,
		Type:   nspAPI.Target_DEFAULT,
		Stream: lb.Stream,
	})
	if err != nil {
		return fmt.Errorf("failed to create target registry watcher (%s): %w",
			lb.Stream.Name, err)
	}
	logger := lb.logger.WithValues("func", "watchTargets")
	for {
		targetResponse, err := watchTarget.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("target registry watcher receive error (%s): %w",
				lb.Stream.Name, err)
		}
		logger.V(2).Info("recv", "response", targetResponse)
		err = lb.setTargets(targetResponse.GetTargets())
		if err != nil {
			logger.Error(err, "setTargets", "targets", targetResponse.GetTargets())
		}
	}
	return nil
}

// TODO/FIXME: setTargets might include a lingering Target with IPs that have
// been alread re-assigned to an interface connecting LB with a proxy. In such
// cases for IPv6 the error returned by FWMark route add attempt was EINVAL
// while for IPV4 it succeeded(!!!): There seems to be sg weird around NSM heal
// that can affect TAPAs causing them to miss proxy loss events and thus NSM heal
// won't kick in when a trench is removed and redeployed: The TAPA gets stuck
// trying to refersh the connection, but it fails due to sticking to an old proxy
// not around anymore (no reselect due to no heal). On one occasion the issue
// resolved after ~40 minutes when the TAPA finally got an NSM DELETE event and
// heal recovered the connection changing its local IPs as well. NSM datapath
// monitoring (even a custom one) would be really helpful IMHO in TAPA, as it
// should lead to NSM heal with reconnect in such cases. (Although, the Target
// announcement should also be synchronized with NSM connection availability in
// the TAPA.)
func (lb *LoadBalancer) setTargets(targets []*nspAPI.Target) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if !lb.isStreamRunning() {
		return nil
	}

	var errFinal error
	var update bool = false
	// to log incoming targets in case of any change compared to local ones
	// (preferably before any update have been performed for better traceability)
	logIncomingTargets := func() {
		if !update {
			update = true
			lb.logger.V(1).Info("change in configuration", "func", "setTargets", "inTargets", targets)
		}
	}
	if len(targets) != (len(lb.targets) + len(lb.pendingTargets)) {
		logIncomingTargets()
	}
	newTargetsMap := make(map[int]types.Target)
	for _, target := range targets {
		t, err := NewTarget(target, lb.netUtils, lb.targetHitsMetrics, lb.IdentifierOffset)
		if err != nil {
			continue
		}
		newTargetsMap[t.GetIdentifier()] = t
	}

	// check pending targets to remove
	for identifier, pendingTarget := range lb.pendingTargets {
		// Remove pending targets not part of the configuration anymore.
		// (Otherwise, a succesfull AddTarget was needed as the prerequisite
		// for a removal. Meaning, a "shadow" Target disrupting load-balancing
		// would be around until a setTargets call could remove it finally.)
		newTarget, exists := newTargetsMap[identifier]
		if !exists {
			logIncomingTargets()
			if err := lb.RemoveTarget(identifier); err != nil && !errors.Is(err, errNoTarget) {
				errFinal = utils.AppendErr(errFinal, err)
			}
			continue
		}
		pendingTargetIPSet := sets.New(pendingTarget.GetIps()...)
		newTargetIPSet := sets.New(newTarget.GetIps()...)
		if !pendingTargetIPSet.Equal(newTargetIPSet) { // ips have changed
			logIncomingTargets()
			if err := lb.RemoveTarget(identifier); err != nil && !errors.Is(err, errNoTarget) {
				errFinal = utils.AppendErr(errFinal, err)
			}
			continue
		}
		// same target in pending list; remove target from new list, there
		// seems to be no reason for an instant add attempt
		// note: pending targets are retried periodically and on interface
		// events
		delete(newTargetsMap, identifier)
	}
	// check targets to remove
	for identifier, target := range lb.targets {
		newTarget, exists := newTargetsMap[identifier]
		if !exists {
			logIncomingTargets()
			err := lb.RemoveTarget(identifier)
			if err != nil {
				errFinal = utils.AppendErr(errFinal, err)
			}
			continue
		}
		targetIPSet := sets.New(target.GetIps()...)
		newTargetIPSet := sets.New(newTarget.GetIps()...)
		if targetIPSet.Equal(newTargetIPSet) { // have the same IPs?
			delete(newTargetsMap, identifier)
			continue
		}
		// Have different IPs, so the target IPs have changed and need to be removed and re-added
		logIncomingTargets()
		err := lb.RemoveTarget(identifier)
		if err != nil {
			errFinal = utils.AppendErr(errFinal, err)
		}
	}
	// check targets to add
	for _, target := range newTargetsMap {
		logIncomingTargets()
		err := lb.AddTarget(target)
		if err != nil {
			errFinal = utils.AppendErr(errFinal, err)
		}
	}
	return errFinal
}

// TODO: revisit if timeout based periodic check can be removed
func (lb *LoadBalancer) processPendingTargets(ctx context.Context) {
	checkf := func() {
		lb.mu.Lock()
		lb.verifyTargets()
		lb.retryPendingTargets()
		lb.mu.Unlock()
	}
	drainf := func(ctx context.Context) bool {
		// drain pendingCh for 100 ms to be protected against bursts
		for {
			select {
			case <-lb.pendingCh:
			case <-time.After(100 * time.Millisecond):
				return true
			case <-ctx.Done():
				return false
			}
		}
	}

	for {
		select {
		case <-time.After(10 * time.Second):
			checkf()
		case <-lb.pendingCh:
			if drainf(ctx) {
				checkf()
			}
		case <-ctx.Done():
			return
		}
	}
}

// triggerPendingTargets -
// Sends trigger to processPendingTargets()
func (lb *LoadBalancer) triggerPendingTargets() {
	lb.mu.Lock()
	if !lb.isStreamRunning() {
		return
	}
	lb.mu.Unlock()

	lb.pendingCh <- struct{}{}
}

func (lb *LoadBalancer) verifyTargets() {
	for _, target := range lb.targets {
		if target.Verify() {
			continue
		}
		lb.logger.Info("removing target after verification", "target", target)
		err := lb.RemoveTarget(target.GetIdentifier())
		if err != nil {
			// note: currently verification checks route availability,
			// thus RemoveTarget shall probably fail with no route error
			lb.logger.Error(err, "delete target after verification", "target", target)
		}
		lb.addPendingTarget(target)
	}
}

func (lb *LoadBalancer) retryPendingTargets() {
	if len(lb.pendingTargets) > 0 {
		lb.logger.V(1).Info("retry pending targets", "pendingLength", len(lb.pendingTargets))
	}
	for _, target := range lb.pendingTargets {
		err := lb.AddTarget(target)
		if err != nil {
			lb.logger.Error(err, "add target from pending list")
		}
	}
}

func (lb *LoadBalancer) addPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if exists {
		return
	}
	lb.logger.Info("add pending target", "func", "addPendingTarget", "target", target)
	lb.pendingTargets[target.GetIdentifier()] = target
}

func (lb *LoadBalancer) removeFromPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if !exists {
		return
	}
	lb.logger.Info("remove pending target", "func", "removeFromPendingTarget", "target", target)
	delete(lb.pendingTargets, target.GetIdentifier())
}

func (lb *LoadBalancer) removeFromPendingTargetByIdentifier(identifier int) {
	target, exists := lb.pendingTargets[identifier]
	if !exists {
		return
	}
	lb.logger.Info("remove pending target by identifier", "target", target)
	delete(lb.pendingTargets, identifier)
}

func (lb *LoadBalancer) isStreamRunning() bool {
	return lb.ctx != nil && lb.ctx.Err() == nil
}

// InterfaceCreated -
// When a new NSM interface of interest appears trigger processPendingTargets()
// to attempt configuring pending Targets (missing route could have become available)
func (lb *LoadBalancer) InterfaceCreated(intf networking.Iface) {
	if strings.HasPrefix(intf.GetName(), GetInterfaceNamePrefix()) { // load-balancer NSE interface
		lb.logger.V(2).Info("InterfaceCreated", "interface", intf.GetName())
		lb.triggerPendingTargets()
	}
}

// InterfaceDeleted -
// When a NSM interface of interest disappears trigger processPendingTargets()
// to verify if configured Targets still have working routes
func (lb *LoadBalancer) InterfaceDeleted(intf networking.Iface) {
	if strings.HasPrefix(intf.GetName(), GetInterfaceNamePrefix()) { // load-balancer NSE interface
		lb.logger.V(2).Info("InterfaceDeleted", "interface", intf.GetName())
		lb.triggerPendingTargets()
	}
}
