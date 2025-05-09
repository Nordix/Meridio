/*
Copyright (c) 2021-2023 Nordix Foundation
Copyright (c) 2025 OpenInfra Foundation Europe

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
	"fmt"
	"io/fs"
	"net"
	"sync"

	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/utils"
	"github.com/vishvananda/netlink"
)

// SourceBasedRoute -
type SourceBasedRoute struct {
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

func (sbr *SourceBasedRoute) removeRoute() error {
	// Delete Route
	src := net.IPv4(0, 0, 0, 0)
	if sbr.family() == netlink.FAMILY_V6 {
		src = net.ParseIP("::")
	}
	route := &netlink.Route{
		Table: sbr.tableID,
		Src:   src,
	}
	if err := netlink.RouteDel(route); err != nil {
		return fmt.Errorf("failed to remove route (%s) for source base routing: %w", route.String(), err)
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
			return fmt.Errorf("nexthop already exists (%s) while adding nexthop: %w", nexthop, fs.ErrExist)
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
	if len(sbr.nexthops) > 0 {
		err = sbr.updateRoute()
	} else {
		err = sbr.removeRoute()
	}
	return err
}

func (sbr *SourceBasedRoute) Delete() error {
	sbr.mu.Lock()
	defer sbr.mu.Unlock()
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
	// routing table might be shared by multiple SourceBasedRoutes, only cleanup route if srb has dedicated nexthops
	if len(sbr.nexthops) > 0 {
		if err := sbr.removeRoute(); err != nil {
			return err
		}
	}
	return nil
}

func (sbr *SourceBasedRoute) family() int {
	if sbr.vip.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
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
	err = sourceBasedRoute.create()
	if err != nil {
		return nil, err
	}
	return sourceBasedRoute, nil
}

// NexthopRoute -
// Manages default multipath routes in a routing table towards multiple nexthops
// for both IPv4 and IPv6. (IPv4 and IPv6 uses the same table ID value)
type NexthopRoute struct {
	tableID  int                     // kernel routing table ID where default multipath route is installed
	nexthops map[int][]*netlink.Addr // store IPv4 and IPv6 nexthops separately
	mu       sync.Mutex
}

// AddNexthop -
// Adds a nexthop which shall be included in the multipath route
func (nr *NexthopRoute) AddNexthop(nexthop string) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	netlinkAddr, err := netlink.ParseAddr(nexthop)
	if err != nil {
		return fmt.Errorf("failed ParseAddr (%s) while adding nexthop: %w", nexthop, err)
	}

	family := getNetlinkFamily(netlinkAddr)
	originalList := nr.nexthops[family]

	// don't append if already exists
	for _, nh := range originalList {
		if netlinkAddr.Equal(*nh) {
			return fmt.Errorf("nexthop already exists (%s) while adding nexthop: %w", nexthop, fs.ErrExist)
		}
	}

	nr.nexthops[family] = append(originalList, netlinkAddr)
	err = nr.updateRoute(family)
	if err != nil {
		nr.nexthops[family] = removeAddrFromList(netlinkAddr, nr.nexthops[family])
		return fmt.Errorf("error configuring nexthop %v: %w", nexthop, err)
	}
	log.Logger.Info("Added nexthop", "nexthop", nexthop, "tableID", nr.tableID)
	return nil
}

// AddNexthop -
// Removes a nexthop from the multipath route
func (nr *NexthopRoute) RemoveNexthop(nexthop string) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	netlinkAddr, err := netlink.ParseAddr(nexthop)
	if err != nil {
		return fmt.Errorf("failed ParseAddr (%s) while removing nexthop: %w", nexthop, err)
	}

	family := getNetlinkFamily(netlinkAddr)
	originalList := nr.nexthops[family]
	nr.nexthops[family] = removeAddrFromList(netlinkAddr, nr.nexthops[family])

	if len(nr.nexthops[family]) < len(originalList) { // Only update route if something was removed
		if len(nr.nexthops[family]) > 0 {
			err = nr.updateRoute(family)
		} else {
			err = nr.removeRoute(family)
		}
	}
	if err != nil {
		return fmt.Errorf("error removing nexthop %v: %w", nexthop, err)
	}

	log.Logger.Info("Removed nexthop", "nexthop", nexthop, "tableID", nr.tableID)
	return nil
}

// Delete -
// Cleans up both IPv4 and IPv6 routes if any
func (nr *NexthopRoute) Delete() error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	var err error
	for family := range nr.nexthops {
		if len(nr.nexthops[family]) > 0 {
			if delErr := nr.removeRoute(family); delErr != nil {
				err = utils.AppendErr(err, delErr)
			}
		}
	}
	return nil
}

func (nr *NexthopRoute) updateRoute(family int) error {
	nexthops := []*netlink.NexthopInfo{}
	for _, nexthop := range nr.nexthops[family] {
		nexthops = append(nexthops, &netlink.NexthopInfo{
			Gw: nexthop.IP,
		})
	}
	src := net.IPv4(0, 0, 0, 0)
	if netlink.FAMILY_V6 == family {
		src = net.ParseIP("::")
	}
	route := &netlink.Route{
		Table:     nr.tableID,
		Src:       src,
		MultiPath: nexthops,
	}
	err := netlink.RouteReplace(route)
	if err != nil {
		return fmt.Errorf("failed to update nexthop route (%s): %w", route.String(), err)
	}
	return nil
}

func (nr *NexthopRoute) removeRoute(family int) error {
	// Delete Route
	src := net.IPv4(0, 0, 0, 0)
	if netlink.FAMILY_V6 == family {
		src = net.ParseIP("::")
	}
	route := &netlink.Route{
		Table: nr.tableID,
		Src:   src,
	}
	if err := netlink.RouteDel(route); err != nil {
		return fmt.Errorf("failed to remove nexthop route (%s): %w", route.String(), err)
	}
	return nil
}

// NewNexthopRoute -
func NewNexthopRoute(tableID int) *NexthopRoute {
	nextHopRoute := &NexthopRoute{
		tableID: tableID,
		nexthops: map[int][]*netlink.Addr{
			netlink.FAMILY_V4: {},
			netlink.FAMILY_V6: {},
		},
	}
	return nextHopRoute
}

func getNetlinkFamily(addr *netlink.Addr) int {
	if addr.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
}

func removeAddrFromList(addrToRemove *netlink.Addr, list []*netlink.Addr) []*netlink.Addr {
	for index, addr := range list {
		if addrToRemove.Equal(*addr) {
			lastIndex := len(list) - 1
			list[index] = list[lastIndex] // Swap with the last element
			return list[:lastIndex]       // Return the slice with the last element removed
		}
	}
	return list // Return the original slice if the element was not found
}
