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
	"github.com/vishvananda/netlink"

	"github.com/nordix/meridio/pkg/log"
)

// FWMarkRoute -
type FWMarkRoute struct {
	ip      *netlink.Addr
	fwmark  int
	tableID int
	route   *netlink.Route
}

// Delete -
func (fwmr *FWMarkRoute) Delete() error {
	rule := netlink.NewRule()
	rule.Table = fwmr.tableID
	rule.Family = fwmr.family()
	err := netlink.RuleDel(rule)
	if err != nil {
		return err
	}

	route := &netlink.Route{
		Gw:    fwmr.ip.IP,
		Table: fwmr.tableID,
	}
	return netlink.RouteDel(route)
}

func (fwmr *FWMarkRoute) Verify() bool {
	routes, err := netlink.RouteListFiltered(fwmr.family(), fwmr.route, netlink.RT_FILTER_GW|netlink.RT_FILTER_TABLE)
	if err != nil {
		log.Logger.V(1).Info("Verify FWMarkRoute", "table", fwmr.tableID, "fwmark", fwmr.fwmark, "error", err)
		return false
	}
	return len(routes) > 0
}

func (fwmr *FWMarkRoute) configure() error {
	_ = fwmr.Delete()

	rule := netlink.NewRule()
	rule.Table = fwmr.tableID
	rule.Mark = fwmr.fwmark
	rule.Family = fwmr.family()
	err := netlink.RuleAdd(rule)
	if err != nil {
		return err
	}

	// Old ARP/NDP entry for that IP address could cause temporary issue.
	fwmr.cleanNeighbor()

	fwmr.route = &netlink.Route{
		Gw:    fwmr.ip.IP,
		Table: fwmr.tableID,
	}
	return netlink.RouteAdd(fwmr.route)
}

func (fwmr *FWMarkRoute) family() int {
	if fwmr.ip.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
}

func (fwmr *FWMarkRoute) cleanNeighbor() {
	neighbors, err := netlink.NeighList(0, 0)
	if err != nil {
		log.Logger.V(1).Info("fetching Neighbor list", "error", err)
		return
	}
	for _, n := range neighbors {
		if n.IP.Equal(fwmr.ip.IP) {
			err = netlink.NeighDel(&n)
			if err != nil {
				log.Logger.V(1).Info("delete from neighbor list", "neighbor", n, "error", err)
			}
		}
	}
}

// NewFWMarkRoute -
func NewFWMarkRoute(ip string, fwmark int, tableID int) (*FWMarkRoute, error) {
	netlinkAddr, err := netlink.ParseAddr(ip)
	if err != nil {
		return nil, err
	}
	fwMarkRoute := &FWMarkRoute{
		ip:      netlinkAddr,
		fwmark:  fwmark,
		tableID: tableID,
	}
	err = fwMarkRoute.configure()
	if err != nil {
		returnErr := err
		err := fwMarkRoute.Delete()
		if err != nil {
			return nil, err
		}
		return nil, returnErr
	}
	return fwMarkRoute, nil
}
