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

package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/loadbalancer/flow"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/sirupsen/logrus"
)

// LoadBalancer -
type LoadBalancer struct {
	*nspAPI.Stream
	TargetRegistryClient       nspAPI.TargetRegistryClient
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
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
}

func New(
	stream *nspAPI.Stream,
	targetRegistryClient nspAPI.TargetRegistryClient,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	m int,
	n int,
	nfqueue int,
	netUtils networking.Utils,
	lbFactory types.NFQueueLoadBalancerFactory,
) (types.Stream, error) {
	nfqlb, err := lbFactory.New(stream.GetName(), m, n)
	if err != nil {
		return nil, err
	}
	loadBalancer := &LoadBalancer{
		Stream:                     stream,
		TargetRegistryClient:       targetRegistryClient,
		ConfigurationManagerClient: configurationManagerClient,
		flows:                      make(map[string]types.Flow),
		nfqlb:                      nfqlb,
		targets:                    make(map[int]types.Target),
		netUtils:                   netUtils,
		nfqueue:                    nfqueue,
		pendingTargets:             make(map[int]types.Target),
		pendingCh:                  make(chan struct{}, 10),
	}
	// first enable kernel's IP defrag except for the interfaces facing targets
	// (defrag is needed by Flows to match rules with L4 information)
	//loadBalancer.defrag, err = NewDefrag(types.InterfaceNamePrefix)
	loadBalancer.defrag, err = NewDefrag(GetInterfaceNamePrefix())
	if err != nil {
		logrus.Warnf("Stream '%v' Defrag setup err=%v", loadBalancer.GetName(), err)
		return nil, err
	}
	err = nfqlb.Start()
	if err != nil {
		return nil, err
	}
	logrus.Infof("Stream '%v' created", loadBalancer.GetName())
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
		logrus.Debugf("Stream '%v' Start watchTargets", lb.GetName())
		err := retry.Do(func() error {
			return lb.watchTargets(lb.ctx)
		}, retry.WithContext(lb.ctx),
			retry.WithDelay(500*time.Millisecond),
			retry.WithErrorIngnored())
		if err != nil {
			logrus.Errorf("Stream '%v' watch Targets err: %v", lb.GetName(), err)
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
	logrus.Infof("Stream '%v' delete", lb.GetName())
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
	err = lb.nfqlb.Activate(target.GetIdentifier())
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
	logrus.Infof("stream: %v, target added: %v", lb.Stream.GetName(), target)
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
	logrus.Infof("stream: %v, target removed: %v", lb.Stream.GetName(), target)
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
		err = lb.setFlows(flowResponse.Flows)
		if err != nil {
			logrus.Warnf("Stream '%v' err set flows: %v", lb.GetName(), err) // todo
		}
	}
	return nil
}

func (lb *LoadBalancer) setFlows(flows []*nspAPI.Flow) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if !lb.isStreamRunning() {
		return nil
	}
	var errFinal error
	remainingFlows := make(map[string]struct{})
	for name := range lb.flows {
		remainingFlows[name] = struct{}{}
	}
	for _, f := range flows {
		fl, exists := lb.flows[f.GetName()]
		if !exists { // create
			newFlow, err := flow.New(f, flow.WithNFQueueLoadBalancer(lb.nfqlb))
			if err != nil {
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
				continue
			}
			lb.flows[f.GetName()] = newFlow
		} else { // update
			err := fl.Update(f)
			if err != nil {
				errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
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
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}
		delete(lb.flows, name)
	}
	// check if flow service can be enabled (needs at least 1 flow)
	// TODO: no flows in any of the streams?
	if len(lb.flows) > 0 {
		health.SetServingStatus(lb.ctx, health.FlowSvc, true)
	}
	return errFinal
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
		logrus.Tracef("Stream '%v' watchTargets: %v", lb.GetName(), targetResponse)
		err = lb.setTargets(targetResponse.GetTargets())
		if err != nil {
			logrus.Warnf("Stream '%v' err set targets: %v", lb.GetName(), err) // todo
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
	toRemoveTargetsMap := make(map[int]struct{})
	for identifier := range lb.targets {
		toRemoveTargetsMap[identifier] = struct{}{}
	}
	logrus.Tracef("Stream '%v' setTargets: %v", lb.GetName(), targets)
	for _, target := range targets { // targets to add
		t, err := NewTarget(target, lb.netUtils)
		if err != nil {
			continue
		}
		if lb.targetExists(t.GetIdentifier()) {
			delete(toRemoveTargetsMap, t.GetIdentifier())
		} else {
			err = lb.AddTarget(t) // todo: pending targets?
			if err != nil {
				errFinal = fmt.Errorf("%w; %v", errFinal, err)
			}
		}
	}
	for identifier := range toRemoveTargetsMap { // targets to remove
		err := lb.RemoveTarget(identifier)
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
			logrus.Warnf("err deleting target (%v): %v", target.GetIdentifier(), err)
		}
		lb.addPendingTarget(target)
	}
}

func (lb *LoadBalancer) retryPendingTargets() {
	for _, target := range lb.pendingTargets {
		err := lb.AddTarget(target)
		if err != nil {
			logrus.Warnf("Stream '%v' err add target (pending): %v", lb.GetName(), err) // todo
		}
	}
}

func (lb *LoadBalancer) addPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if exists {
		return
	}
	logrus.Infof("Stream '%v' add pending target: %v", lb.GetName(), target)
	lb.pendingTargets[target.GetIdentifier()] = target
}

func (lb *LoadBalancer) removeFromPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if !exists {
		return
	}
	logrus.Infof("Stream '%v' remove pending target: %v", lb.GetName(), target)
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
		logrus.Tracef("Stream '%v' InterfaceCreated %v", lb.GetName(), intf.GetName())
		lb.triggerPendingTargets()
	}
}

// InterfaceDeleted -
// When a NSM interface of interest disappears trigger processPendingTargets()
// to verify if configured Targets still have working routes
func (lb *LoadBalancer) InterfaceDeleted(intf networking.Iface) {
	if strings.HasPrefix(intf.GetName(), GetInterfaceNamePrefix()) { // load-balancer NSE interface
		logrus.Tracef("Stream '%v' InterfaceDeleted %v", lb.GetName(), intf.GetName())
		lb.triggerPendingTargets()
	}
}
