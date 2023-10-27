/*
Copyright (c) 2021-2023 Nordix Foundation

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
	"github.com/nordix/meridio/pkg/loadbalancer/flow"
	targetMetrics "github.com/nordix/meridio/pkg/loadbalancer/target"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/retry"
	"k8s.io/apimachinery/pkg/util/sets"
)

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
) (types.Stream, error) {
	n := int(stream.GetMaxTargets())
	m := int(stream.GetMaxTargets()) * 100
	nfqlb, err := lbFactory.New(stream.GetName(), m, n)
	if err != nil {
		return nil, err
	}

	logger := log.Logger.WithValues("class", "LoadBalancer", "instance", stream.GetName())
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
	}
	// first enable kernel's IP defrag except for the interfaces facing targets
	// (defrag is needed by Flows to match rules with L4 information)
	//loadBalancer.defrag, err = NewDefrag(types.InterfaceNamePrefix)
	loadBalancer.defrag, err = NewDefrag(GetInterfaceNamePrefix())
	if err != nil {
		logger.Error(err, "Defrag setup")
		return nil, err
	}
	err = nfqlb.Start()
	if err != nil {
		return nil, err
	}
	logger.Info("Created")
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
		lb.logger.V(1).Info("Start watchTargets")
		err := retry.Do(func() error {
			return lb.watchTargets(lb.ctx)
		}, retry.WithContext(lb.ctx),
			retry.WithDelay(500*time.Millisecond),
			retry.WithErrorIngnored())
		if err != nil {
			lb.logger.Error(err, "watchTargets")
		}
	}()
	go lb.processPendingTargets(lb.ctx)
	err := retry.Do(func() error {
		return lb.watchFlows(lb.ctx)
	}, retry.WithContext(lb.ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		return err
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
			errFinal = fmt.Errorf("%w; target: %v", errFinal, err)
		}
	}
	for _, flow := range lb.flows {
		err := flow.Delete()
		if err != nil {
			errFinal = fmt.Errorf("%w; flow: %v", errFinal, err)
		}
	}
	err := lb.nfqlb.Delete()
	if err != nil {
		errFinal = fmt.Errorf("%w; nfqlb: %v", errFinal, err)
	}
	lb.logger.Info("Delete")
	return errFinal // todo
}

// AddTarget -
func (lb *LoadBalancer) AddTarget(target types.Target) error {
	exists := lb.targetExists(target.GetIdentifier())
	if exists {
		return errors.New("the target is already registered")
	}
	err := target.Configure() // TODO: avoid multiple identical ip rule entries (e.g. after container crash)
	if err != nil {
		lb.addPendingTarget(target)
		returnErr := err
		err = target.Delete()
		if err != nil {
			return fmt.Errorf("%w; %v", err, returnErr)
		}
		return returnErr
	}
	err = lb.nfqlb.Activate(target.GetIdentifier(), target.GetIdentifier()+lb.IdentifierOffset)
	if err != nil {
		lb.addPendingTarget(target)
		returnErr := err
		err = target.Delete()
		if err != nil {
			return fmt.Errorf("%w; %v", err, returnErr)
		}
		return returnErr
	}
	lb.targets[target.GetIdentifier()] = target
	lb.removeFromPendingTarget(target)
	lb.logger.Info("AddTarget", "target", target)
	return nil
}

// RemoveTarget -
func (lb *LoadBalancer) RemoveTarget(identifier int) error {
	target := lb.getTarget(identifier)
	if target == nil {
		return errors.New("the target is not existing")
	}
	var errFinal error
	lb.removeFromPendingTarget(target)
	err := lb.nfqlb.Deactivate(target.GetIdentifier())
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
	}
	err = target.Delete()
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
	}
	delete(lb.targets, target.GetIdentifier())
	lb.logger.Info("RemoveTarget", "target", target)
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
		return err
	}
	for {
		flowResponse, err := watchFlow.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
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
func (lb *LoadBalancer) watchTargets(ctx context.Context) error {
	watchTarget, err := lb.TargetRegistryClient.Watch(ctx, &nspAPI.Target{
		Status: nspAPI.Target_ENABLED,
		Type:   nspAPI.Target_DEFAULT,
		Stream: lb.Stream,
	})
	if err != nil {
		return err
	}
	for {
		targetResponse, err := watchTarget.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		lb.logger.V(2).Info("watchTargets", "response", targetResponse)
		err = lb.setTargets(targetResponse.GetTargets())
		if err != nil {
			lb.logger.Error(err, "setTargets")
		}
	}
	return nil
}

func (lb *LoadBalancer) setTargets(targets []*nspAPI.Target) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if !lb.isStreamRunning() {
		return nil
	}
	var errFinal error
	newTargetsMap := make(map[int]types.Target)
	for _, target := range targets {
		t, err := NewTarget(target, lb.netUtils, lb.targetHitsMetrics, lb.IdentifierOffset)
		if err != nil {
			continue
		}
		newTargetsMap[t.GetIdentifier()] = t
	}
	for identifier, target := range lb.targets { // targets to remove
		newTarget, exists := newTargetsMap[identifier]
		if !exists {
			err := lb.RemoveTarget(identifier)
			if err != nil {
				errFinal = fmt.Errorf("%w; %v", errFinal, err)
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
		err := lb.RemoveTarget(identifier)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
		}
	}
	for _, target := range newTargetsMap { // targets to add
		err := lb.AddTarget(target)
		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err)
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
		err := lb.RemoveTarget(target.GetIdentifier())
		if err != nil {
			lb.logger.Error(err, "deleting target", "target", target)
		}
		lb.addPendingTarget(target)
	}
}

func (lb *LoadBalancer) retryPendingTargets() {
	for _, target := range lb.pendingTargets {
		err := lb.AddTarget(target)
		if err != nil {
			lb.logger.Error(err, "add target (pending)")
		}
	}
}

func (lb *LoadBalancer) addPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if exists {
		return
	}
	lb.logger.Info("add pending target", "target", target)
	lb.pendingTargets[target.GetIdentifier()] = target
}

func (lb *LoadBalancer) removeFromPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if !exists {
		return
	}
	lb.logger.Info("remove pending target", "target", target)
	delete(lb.pendingTargets, target.GetIdentifier())
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
