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

package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/nfqlb"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

// LoadBalancer -
type LoadBalancer struct {
	*nspAPI.Stream
	TargetRegistryClient       nspAPI.TargetRegistryClient
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	nfqlb                      types.NFQueueLoadBalancer
	vips                       []*virtualIP
	targets                    map[int]types.Target // key: Identifier
	netUtils                   networking.Utils
	nfqueue                    int
	mu                         sync.Mutex
	cancel                     context.CancelFunc
	pendingTargets             map[int]types.Target // key: Identifier
}

func New(stream *nspAPI.Stream, targetRegistryClient nspAPI.TargetRegistryClient, configurationManagerClient nspAPI.ConfigurationManagerClient, m int, n int, nfqueue int, netUtils networking.Utils) (types.Stream, error) {
	nfqlb, err := nfqlb.New(stream.GetName(), m, n, nfqueue)
	if err != nil {
		return nil, err
	}
	loadBalancer := &LoadBalancer{
		Stream:                     stream,
		TargetRegistryClient:       targetRegistryClient,
		ConfigurationManagerClient: configurationManagerClient,
		vips:                       []*virtualIP{},
		nfqlb:                      nfqlb,
		targets:                    make(map[int]types.Target),
		netUtils:                   netUtils,
		nfqueue:                    nfqueue,
		pendingTargets:             make(map[int]types.Target),
	}
	err = nfqlb.Start()
	if err != nil {
		return nil, err
	}
	logrus.Infof("Stream '%v' created", loadBalancer.GetName())
	return loadBalancer, nil
}

func (lb *LoadBalancer) Start(ctx context.Context) error {
	var c context.Context
	c, lb.cancel = context.WithCancel(ctx)
	go func() { // todo
		err := lb.watchTargets(c)
		if err != nil {
			logrus.Errorf("watch Targets err: %v", err)
		}
	}()
	go lb.processPendingTargets(c)
	err := lb.watchFlows(c)
	if err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) Delete() error {
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
	for _, vip := range lb.vips {
		err := vip.Delete()
		if err != nil {
			errFinal = fmt.Errorf("%w; vip: %v", errFinal, err)
		}
	}
	err := lb.nfqlb.Delete()
	if err != nil {
		errFinal = fmt.Errorf("%w; nfqlb: %v", errFinal, err)
	}
	return errFinal // todo
}

// AddTarget -
func (lb *LoadBalancer) AddTarget(target types.Target) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	exists := lb.targetExists(target.GetIdentifier())
	if exists {
		return errors.New("the target is already registered")
	}
	err := target.Configure()
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
	lb.mu.Lock()
	defer lb.mu.Unlock()
	target := lb.getTarget(identifier)
	if target == nil {
		return errors.New("the target is not existing")
	}
	lb.removeFromPendingTarget(target)
	err := lb.nfqlb.Deactivate(target.GetIdentifier())
	if err != nil {
		return err
	}
	err = target.Delete()
	if err != nil {
		return err
	}
	delete(lb.targets, target.GetIdentifier())
	logrus.Infof("stream: %v, target removed: %v", lb.Stream.GetName(), target)
	return nil
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

func (lb *LoadBalancer) SetVIPs(vips []string) error {
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range lb.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, lb.nfqueue, lb.netUtils)
			if err != nil {
				logrus.Errorf("Error adding VIP: %v", err)
				continue
			}
			lb.vips = append(lb.vips, newVIP)
		}
		delete(currentVIPs, vip)
	}
	// delete remaining vips
	for index := 0; index < len(lb.vips); index++ {
		vip := lb.vips[index]
		if _, ok := currentVIPs[vip.prefix]; ok {
			lb.vips = append(lb.vips[:index], lb.vips[index+1:]...)
			index--
			err := vip.Delete()
			if err != nil {
				logrus.Errorf("Error deleting vip: %v", err)
			}
		}
	}
	return nil
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
		vips := []string{}
		for _, flow := range flowResponse.Flows {
			for _, vip := range flow.Vips {
				vips = append(vips, vip.Address)
			}
		}
		err = lb.SetVIPs(vips)
		if err != nil {
			return err
		}
	}
	return nil
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
		err = lb.setTargets(targetResponse.GetTargets())
		if err != nil {
			logrus.Warnf("err set targets: %v", err) // todo
		}
	}
	return nil
}

func (lb *LoadBalancer) setTargets(targets []*nspAPI.Target) error {
	var errFinal error
	toRemoveTargetsMap := make(map[int]struct{})
	for identifier := range lb.targets {
		toRemoveTargetsMap[identifier] = struct{}{}
	}
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

// todo: find a better way to detect when the routes are available for the pending targets
func (lb *LoadBalancer) processPendingTargets(ctx context.Context) {
	for {
		select {
		case <-time.After(10 * time.Second):
			lb.retryPendingTargets()
		case <-ctx.Done():
			return
		}
	}
}

func (lb *LoadBalancer) retryPendingTargets() {
	for _, target := range lb.pendingTargets {
		err := lb.AddTarget(target)
		if err != nil {
			logrus.Warnf("err add target (pending): %v", err) // todo
		}
	}
}

func (lb *LoadBalancer) addPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if exists {
		return
	}
	logrus.Infof("add pending target: %v", target)
	lb.pendingTargets[target.GetIdentifier()] = target
}

func (lb *LoadBalancer) removeFromPendingTarget(target types.Target) {
	_, exists := lb.pendingTargets[target.GetIdentifier()]
	if !exists {
		return
	}
	logrus.Infof("remove pending target: %v", target)
	delete(lb.pendingTargets, target.GetIdentifier())
}
