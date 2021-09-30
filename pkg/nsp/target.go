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
	"errors"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/sirupsen/logrus"
)

type TargetRegistry interface {
	Add(*nspAPI.Target) error
	Remove(*nspAPI.Target) (*nspAPI.Target, error)
	Get(nspAPI.Target_Type) []*nspAPI.Target
	Update(target *nspAPI.Target) error
	Exists(*nspAPI.Target) bool
}

type TargetContextType int32

const (
	_ TargetContextType = iota
	Identifier
)

func (t TargetContextType) String() string {
	switch t {
	case Identifier:
		return "identifier"
	default:
		return "unknown"
	}
}

type targetList struct {
	targets map[nspAPI.Target_Type][]*nspAPI.Target
	mu      sync.Mutex
}

func (tl *targetList) Add(target *nspAPI.Target) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	if tl.exists(target) {
		logrus.Debugf("targetList: target exists: %v", target)
		return errors.New("target already exists")
	}
	logrus.Debugf("targetList: Add target %v", target)
	targetType := target.GetType()
	tl.targets[targetType] = append(tl.targets[targetType], target)
	return nil
}

func (tl *targetList) Remove(target *nspAPI.Target) (*nspAPI.Target, error) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	index := tl.getIndex(target)
	if index < 0 {
		return nil, errors.New("target is not existing")
	}
	logrus.Debugf("targetList: Remove target %v", target)
	targetType := target.GetType()
	t := tl.targets[targetType][index]
	tl.removeIndex(index, targetType)
	if len(tl.targets[targetType]) == 0 {
		delete(tl.targets, targetType)
	}
	return t, nil
}

func (tl *targetList) Get(targetType nspAPI.Target_Type) []*nspAPI.Target {
	return tl.targets[targetType]
}

func (tl *targetList) Update(target *nspAPI.Target) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	if !tl.exists(target) {
		return errors.New("target is not existing")
	}
	index := tl.getIndex(target)
	targetType := target.GetType()
	t := tl.targets[targetType][index]
	t.Context = target.GetContext()
	t.Status = target.GetStatus()
	return nil
}

func (tl *targetList) Exists(target *nspAPI.Target) bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.exists(target)
}

func (tl *targetList) exists(target *nspAPI.Target) bool {
	return tl.getIndex(target) >= 0
}

func (tl *targetList) getIndex(target *nspAPI.Target) int {
	targets := tl.targets[target.GetType()]
	for index, t := range targets {
		if Equals(t, target) {
			return index
		}
	}
	return -1
}

func (tl *targetList) removeIndex(index int, targetType nspAPI.Target_Type) {
	_, exists := tl.targets[targetType]
	if !exists {
		return
	}
	tl.targets[targetType] = append(tl.targets[targetType][:index], tl.targets[targetType][index+1:]...)
}

func Equals(t1 *nspAPI.Target, t2 *nspAPI.Target) bool {
	if (t1.Ips == nil) != (t2.Ips == nil) {
		return false
	}
	if len(t1.Ips) != len(t2.Ips) {
		return false
	}
	ips := map[string]struct{}{}
	for _, ip := range t1.Ips {
		ips[ip] = struct{}{}
	}
	for _, ip := range t2.Ips {
		if _, ok := ips[ip]; !ok {
			return false
		}
	}
	return true
}
