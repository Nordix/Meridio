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
	vips     []*virtualIP
	targets  map[int]*Target // key: Identifier
	netUtils networking.Utils
}

// Start -
func (lb *LoadBalancer) Start() error {
	return exec.Command("nfqlb", "lb", "--qlength=1024").Start()
}

// AddTarget -
func (lb *LoadBalancer) AddTarget(target *Target) error {
	if lb.TargetExists(target) {
		return errors.New("the target is already existing")
	}
	err := target.Configure(lb.netUtils)
	if err != nil {
		return err
	}
	err = lb.activateIdentifier(target.identifier)
	if err != nil {
		returnErr := err
		err = target.Delete()
		if err != nil {
			return fmt.Errorf("%w; target.Delete: %v", err, target.identifier)
		}
		return fmt.Errorf("%w; activateIdentifier: %v", returnErr, target.identifier)
	}
	lb.targets[target.identifier] = target
	return nil
}

// RemoveTarget -
func (lb *LoadBalancer) RemoveTarget(target *Target) error {
	if !lb.TargetExists(target) {
		return errors.New("the target does not exist")
	}
	t := lb.targets[target.identifier]
	err := t.Delete()
	if err != nil {
		return err
	}
	err = lb.deactivateIdentifier(target.identifier)
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
	for _, target := range lb.targets {
		targets = append(targets, target)
	}
	return targets
}

func (lb *LoadBalancer) activateIdentifier(identifier int) error {
	_, err := exec.Command("nfqlb", "activate", fmt.Sprintf("--lookup=%s", strconv.Itoa(identifier)), strconv.Itoa(identifier)).Output()
	return err
}

func (lb *LoadBalancer) deactivateIdentifier(identifier int) error {
	_, err := exec.Command("nfqlb", "deactivate", fmt.Sprintf("--lookup=%s", strconv.Itoa(identifier))).Output()
	return err
}

func (lb *LoadBalancer) configure() error {
	_, err := exec.Command("nfqlb",
		"init",
		"--ownfw=0",
		fmt.Sprintf("--M=%s", strconv.Itoa(lb.m)),
		fmt.Sprintf("--N=%s", strconv.Itoa(lb.n))).Output()
	if err != nil {
		return err
	}
	err = lb.desactivateAll()
	if err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) desactivateAll() error {
	for i := 0; i < lb.n; i++ {
		err := lb.deactivateIdentifier(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lb *LoadBalancer) SetVIPs(vips []string) {
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range lb.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, lb.netUtils)
			if err != nil {
				logrus.Errorf("Load Balancer: Error adding SourceBaseRoute: %v", err)
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
				logrus.Errorf("Load Balancer: Error deleting vip: %v", err)
			}
		}
	}
}

func NewLoadBalancer(vips []string, m int, n int, netUtils networking.Utils) (*LoadBalancer, error) {
	loadBalancer := &LoadBalancer{
		m:        m,
		n:        n,
		vips:     []*virtualIP{},
		targets:  make(map[int]*Target),
		netUtils: netUtils,
	}
	loadBalancer.SetVIPs(vips)
	err := loadBalancer.configure()
	if err != nil {
		return nil, err
	}
	return loadBalancer, nil
}
