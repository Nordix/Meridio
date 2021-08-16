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
)

// FWMarkRoute -
type FWMarkRoute struct {
	ip      *netlink.Addr
	fwmark  int
	tableID int
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

func (fwmr *FWMarkRoute) configure() error {
	rule := netlink.NewRule()
	rule.Table = fwmr.tableID
	rule.Mark = fwmr.fwmark
	rule.Family = fwmr.family()
	err := netlink.RuleAdd(rule)
	if err != nil {
		return err
	}

	route := &netlink.Route{
		Gw:    fwmr.ip.IP,
		Table: fwmr.tableID,
	}
	return netlink.RouteAdd(route)
}

func (fwmr *FWMarkRoute) family() int {
	if fwmr.ip.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
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
