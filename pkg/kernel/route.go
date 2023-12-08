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

package kernel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
)

// SourceBasedRoute -
type SourceBasedRoute struct {
	ctx      context.Context
	cancel   context.CancelFunc
	tableID  int
	vip      *netlink.Addr
	nexthops []*netlink.Addr
	mu       sync.Mutex
}

func (sbr *SourceBasedRoute) create() error {
	rule := netlink.NewRule()
	rule.Table = sbr.tableID
	rule.Src = &net.IPNet{
		IP:   sbr.vip.IP,
		Mask: sbr.vip.Mask,
	}
	rule.Family = sbr.family()
	err := netlink.RuleAdd(rule)
	if err != nil {
		return fmt.Errorf("failed to add rule (%s) for source base routing: %w", rule.String(), err)
	}

	return nil
}

func (sbr *SourceBasedRoute) updateRoute() error {
	nexthops := []*netlink.NexthopInfo{}
	for _, nexthop := range sbr.nexthops {
		nexthops = append(nexthops, &netlink.NexthopInfo{
			Gw: nexthop.IP,
		})
	}
	src := net.IPv4(0, 0, 0, 0)
	if sbr.family() == netlink.FAMILY_V6 {
		src = net.ParseIP("::")
	}
	route := &netlink.Route{
		Table:     sbr.tableID,
		Src:       src,
		MultiPath: nexthops,
	}
	err := netlink.RouteReplace(route)
	if err != nil {
		return fmt.Errorf("failed to update route (%s) for source base routing: %w", route.String(), err)
	}

	return nil
}

func (sbr *SourceBasedRoute) removeFromList(nexthop *netlink.Addr) {
	for index, current := range sbr.nexthops {
		if nexthop.Equal(*current) {
			sbr.nexthops = append(sbr.nexthops[:index], sbr.nexthops[index+1:]...)
			return
		}
	}
}

// AddNexthop -
func (sbr *SourceBasedRoute) AddNexthop(nexthop string) error {
	sbr.mu.Lock()
	defer sbr.mu.Unlock()
	netlinkAddr, err := netlink.ParseAddr(nexthop)
	if err != nil {
		return fmt.Errorf("failed ParseAddr (%s) while adding nexthop: %w", nexthop, err)
	}

	// don't append if already exists
	for _, nh := range sbr.nexthops {
		if netlinkAddr.Equal(*nh) {
			return nil
		}
	}

	sbr.nexthops = append(sbr.nexthops, netlinkAddr)
	err = sbr.updateRoute()
	if err != nil {
		sbr.removeFromList(netlinkAddr)
		return err
	}
	return err
}

// RemoveNexthop -
func (sbr *SourceBasedRoute) RemoveNexthop(nexthop string) error {
	sbr.mu.Lock()
	defer sbr.mu.Unlock()
	netlinkAddr, err := netlink.ParseAddr(nexthop)
	if err != nil {
		return fmt.Errorf("failed ParseAddr (%s) while removing nexthop: %w", nexthop, err)
	}
	sbr.removeFromList(netlinkAddr)
	err = sbr.updateRoute()
	return err
}

func (sbr *SourceBasedRoute) Delete() error {
	sbr.mu.Lock()
	defer sbr.mu.Unlock()
	if sbr.cancel != nil {
		sbr.cancel()
	}
	// Delete Rule
	rule := netlink.NewRule()
	rule.Table = sbr.tableID
	rule.Src = &net.IPNet{
		IP:   sbr.vip.IP,
		Mask: sbr.vip.Mask,
	}
	rule.Family = sbr.family()
	err := netlink.RuleDel(rule)
	if err != nil {
		return fmt.Errorf("failed RuleDel (%s) while deleting source base route: %w", rule.String(), err)
	}
	// Delete Route
	src := net.IPv4(0, 0, 0, 0)
	if sbr.family() == netlink.FAMILY_V6 {
		src = net.ParseIP("::")
	}
	route := &netlink.Route{
		Table: sbr.tableID,
		Src:   src,
	}
	err = netlink.RouteDel(route)
	if err != nil {
		return fmt.Errorf("failed RouteDel (%s) while deleting source base route: %w", route.String(), err)
	}

	return nil
}

func (sbr *SourceBasedRoute) family() int {
	if sbr.vip.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
}

// Todo
// pkg/loadbalancer/stream/loadbalancer.go - processPendingTargets
func (sbr *SourceBasedRoute) verify() {
	for {
		select {
		case <-time.After(10 * time.Second):
			sbr.mu.Lock()
			_ = sbr.updateRoute()
			sbr.mu.Unlock()
		case <-sbr.ctx.Done():
			return
		}
	}
}

// NewSourceBasedRoute -
func NewSourceBasedRoute(tableID int, vip string) (*SourceBasedRoute, error) {
	netlinkAddr, err := netlink.ParseAddr(vip)
	if err != nil {
		return nil, fmt.Errorf("failed ParseAddr (%s) while creating source base route: %w", vip, err)
	}
	sourceBasedRoute := &SourceBasedRoute{
		tableID:  tableID,
		vip:      netlinkAddr,
		nexthops: []*netlink.Addr{},
	}
	sourceBasedRoute.ctx, sourceBasedRoute.cancel = context.WithCancel(context.TODO())
	err = sourceBasedRoute.create()
	if err != nil {
		return nil, err
	}
	go sourceBasedRoute.verify()
	return sourceBasedRoute, nil
}
