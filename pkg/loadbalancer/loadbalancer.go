package loadbalancer

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// LoadBalancer -
type LoadBalancer struct {
	m       int
	n       int
	nfQueue *networking.NFQueue
	vip     *netlink.Addr
	targets map[int]*configuredTarget // key: Identifier
}

type configuredTarget struct {
	target *Target
	fwMark *networking.FWMarkRoute
}

// Start -
func (lb *LoadBalancer) Start() error {
	return exec.Command("lb", "run", "-p").Start()
}

// AddTarget -
func (lb *LoadBalancer) AddTarget(target *Target) error {
	if lb.TargetExists(target) == true {
		return errors.New("The target is already existing.")
	}
	fwMark, err := networking.NewFWMarkRoute(target.ip, target.identifier, target.identifier)
	if err != nil {
		return err
	}
	err = lb.activateIdentifier(target.identifier)
	if err != nil {
		return fmt.Errorf("%w; activateIdentifier: %v", err, target.identifier)
	}
	lb.targets[target.identifier] = &configuredTarget{
		target: target,
		fwMark: fwMark,
	}
	return nil
}

// RemoveTarget -
func (lb *LoadBalancer) RemoveTarget(target *Target) error {
	if lb.TargetExists(target) == false {
		return errors.New("The target does not exist.")
	}
	configuredTarget, _ := lb.targets[target.identifier]
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

func (lb *LoadBalancer) activateIdentifier(identifier int) error {
	_, err := exec.Command("lb", "activate", strconv.Itoa(identifier)).Output()
	return err
}

func (lb *LoadBalancer) desactivateIdentifier(identifier int) error {
	_, err := exec.Command("lb", "deactivate", strconv.Itoa(identifier)).Output()
	return err
}

func (lb *LoadBalancer) configure() {
	_, err := exec.Command("lb",
		"create",
		strconv.Itoa(lb.m),
		strconv.Itoa(lb.n)).Output()
	lb.desactivateAll()

	nfqueue, err := networking.NewNFQueue(lb.vip, 2)
	if err != nil {
		logrus.Errorf("Load Balancer: error configuring nfqueue (iptables): %v", err)
	}
	lb.nfQueue = nfqueue
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

func NewLoadBalancer(vip *netlink.Addr, m int, n int) *LoadBalancer {
	loadBalancer := &LoadBalancer{
		m:       m,
		n:       n,
		vip:     vip,
		targets: make(map[int]*configuredTarget),
	}
	loadBalancer.configure()
	return loadBalancer
}
