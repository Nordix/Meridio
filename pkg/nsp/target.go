package nsp

import (
	"errors"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp"
)

type targetList struct {
	targets []*target
	mu      sync.Mutex
}

type target struct {
	*nspAPI.Target
}

func (t *target) Equals(t2 *target) bool {
	if (t.Ips == nil) != (t2.Ips == nil) {
		return false
	}
	if len(t.Ips) != len(t2.Ips) {
		return false
	}
	ips := map[string]struct{}{}
	for _, ip := range t.Ips {
		ips[ip] = struct{}{}
	}
	for _, ip := range t2.Ips {
		if _, ok := ips[ip]; !ok {
			return false
		}
	}
	return true
}

func (tl *targetList) Exists(nspAPITarget *nspAPI.Target) bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	nt := &target{
		nspAPITarget,
	}
	return tl.exists(nt)
}

func (tl *targetList) exists(tar *target) bool {
	for _, t := range tl.targets {
		if t.Equals(tar) {
			return true
		}
	}
	return false
}

func (tl *targetList) getIndex(tar *target) int {
	for index, t := range tl.targets {
		if t.Equals(tar) {
			return index
		}
	}
	return -1
}

func (tl *targetList) removeIndex(index int) {
	tl.targets = append(tl.targets[:index], tl.targets[index+1:]...)
}

func (tl *targetList) Add(nspAPITarget *nspAPI.Target) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	nt := &target{
		nspAPITarget,
	}
	if tl.exists(nt) {
		return errors.New("target already exists")
	}
	tl.targets = append(tl.targets, nt)
	return nil
}

func (tl *targetList) Remove(nspAPITarget *nspAPI.Target) (*nspAPI.Target, error) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	nt := &target{
		nspAPITarget,
	}
	index := tl.getIndex(nt)
	if index < 0 {
		return nil, errors.New("target is not existing")
	}
	target := tl.targets[index]
	tl.removeIndex(index)
	return target.Target, nil
}

func (tl *targetList) Get() []*nspAPI.Target {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	targets := []*nspAPI.Target{}
	for _, t := range tl.targets {
		targets = append(targets, t.Target)
	}
	return targets
}
