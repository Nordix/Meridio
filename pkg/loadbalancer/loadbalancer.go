package loadbalancer

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

// LoadBalancer -
type LoadBalancer struct {
	m        int
	n        int
	nfQueue  networking.NFQueue
	vip      string
	targets  map[int]*configuredTarget // key: Identifier
	netUtils networking.Utils
}

type configuredTarget struct {
	target *Target
	fwMark networking.FWMarkRoute
}

// Start -
func (lb *LoadBalancer) Start() error {
	return exec.Command("lb", "run", "-p").Start()
}

// AddTarget -
func (lb *LoadBalancer) AddTarget(target *Target) error {
	if lb.TargetExists(target) {
		return errors.New("The target is already existing.")
	}
	fwMark, err := lb.netUtils.NewFWMarkRoute(target.ip, target.identifier, target.identifier)
	if err != nil {
		return err
	}
	err = lb.activateIdentifier(target.identifier)
	if err != nil {
		returnErr := err
		err := fwMark.Delete()
		if err != nil {
			return fmt.Errorf("%w; activateIdentifier fwMark.Delete: %v", err, target.identifier)
		}
		return fmt.Errorf("%w; activateIdentifier: %v", returnErr, target.identifier)
	}
	lb.targets[target.identifier] = &configuredTarget{
		target: target,
		fwMark: fwMark,
	}
	return nil
}

// RemoveTarget -
func (lb *LoadBalancer) RemoveTarget(target *Target) error {
	if !lb.TargetExists(target) {
		return errors.New("The target does not exist.")
	}
	configuredTarget := lb.targets[target.identifier]
	err := configuredTarget.fwMark.Delete()
	if err != nil {
		return err
	}
	err = lb.desactivateIdentifier(target.identifier)
	if err != nil {
		return err
	}
	delete(lb.targets, target.identifier)
	return nil
}

// TargetExists -
func (lb *LoadBalancer) TargetExists(target *Target) bool {
	_, exists := lb.targets[target.identifier]
	return exists
}

// TargetExists -
func (lb *LoadBalancer) GetTargets() []*Target {
	targets := []*Target{}
	for _, configuredTarget := range lb.targets {
		targets = append(targets, configuredTarget.target)
	}
	return targets
}

func (lb *LoadBalancer) activateIdentifier(identifier int) error {
	_, err := exec.Command("lb", "activate", strconv.Itoa(identifier)).Output()
	return err
}

func (lb *LoadBalancer) desactivateIdentifier(identifier int) error {
	_, err := exec.Command("lb", "deactivate", strconv.Itoa(identifier)).Output()
	return err
}

func (lb *LoadBalancer) configure() error {
	_, err := exec.Command("lb",
		"create",
		strconv.Itoa(lb.m),
		strconv.Itoa(lb.n)).Output()
	if err != nil {
		return err
	}
	err = lb.desactivateAll()
	if err != nil {
		return err
	}
	nfqueue, err := lb.netUtils.NewNFQueue(lb.vip, 2)
	if err != nil {
		logrus.Errorf("Load Balancer: error configuring nfqueue (iptables): %v", err)
		return err
	}
	lb.nfQueue = nfqueue
	return nil
}

func (lb *LoadBalancer) desactivateAll() error {
	for i := 1; i <= lb.n; i++ {
		err := lb.desactivateIdentifier(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewLoadBalancer(vip string, m int, n int, netUtils networking.Utils) (*LoadBalancer, error) {
	loadBalancer := &LoadBalancer{
		m:        m,
		n:        n,
		vip:      vip,
		targets:  make(map[int]*configuredTarget),
		netUtils: netUtils,
	}
	err := loadBalancer.configure()
	if err != nil {
		return nil, err
	}
	return loadBalancer, nil
}
