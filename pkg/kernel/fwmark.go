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
	"errors"
	"fmt"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/vishvananda/netlink"

	"github.com/nordix/meridio/pkg/log"
)

// FWMarkRouteError represents an error occurred during FWMarkRoute route operation
type FWMarkRouteError struct {
	msg string
}

func (e *FWMarkRouteError) Error() string {
	return fmt.Sprintf("fwmark route error: %s", e.msg)
}

// FWMarkRouteRuleError represents an error occurred during FWMarkRoute rule operation
type FWMarkRouteRuleError struct {
	msg string
}

func (e *FWMarkRouteRuleError) Error() string {
	return fmt.Sprintf("fwmark route rule configuration error: %s", e.msg)
}

// indicates an error returned by FWMarkRoute where rule is missing during delete
func isErrNoRule(err error) bool {
	return asErrFWMarkRule(err) && errors.Is(err, syscall.ENOENT)
}

// indicates an error returned by FWMarkRoute where route is missing during delete
func isErrNoRoute(err error) bool {
	return asErrFWMarkRoute(err) && errors.Is(err, syscall.ESRCH)
}

// indicates an error returned by FWMarkRoute that is related to a rule operation
func asErrFWMarkRule(err error) bool {
	var fWMarkRouteRuleError *FWMarkRouteRuleError
	return errors.As(err, &fWMarkRouteRuleError)
}

// indicates an error returned by FWMarkRoute that is related to a route operation
func asErrFWMarkRoute(err error) bool {
	var fWMarkRouteError *FWMarkRouteError
	return errors.As(err, &fWMarkRouteError)
}

// FWMarkRoute -
type FWMarkRoute struct {
	ip      *netlink.Addr
	fwmark  int
	tableID int
	route   *netlink.Route
	logger  logr.Logger
}

// Delete -
func (fwmr *FWMarkRoute) Delete() error {
	rule := netlink.NewRule()
	rule.Table = fwmr.tableID
	rule.Family = fwmr.family()
	err := netlink.RuleDel(rule)
	if err != nil {
		return fmt.Errorf("%w:%w",
			&FWMarkRouteRuleError{msg: fmt.Sprintf("failed deleting rule (%s) while deleting fwmark route", rule.String())},
			err)
	}

	route := &netlink.Route{
		Gw:    fwmr.ip.IP,
		Table: fwmr.tableID,
	}
	err = netlink.RouteDel(route)
	if err != nil {
		return fmt.Errorf("%w: %w",
			&FWMarkRouteError{msg: fmt.Sprintf("failed deleting route (%s) while deleting fwmark route", route.String())},
			err)
	}

	return nil
}

func (fwmr *FWMarkRoute) Verify() bool {
	routes, err := netlink.RouteListFiltered(fwmr.family(), fwmr.route, netlink.RT_FILTER_GW|netlink.RT_FILTER_TABLE)
	if err != nil {
		fwmr.logger.V(1).Info("Verify FWMarkRoute", "error", err)
		return false
	}
	return len(routes) > 0
}

func (fwmr *FWMarkRoute) configure() error {
	_ = fwmr.Delete()

	rule := netlink.NewRule()
	rule.Table = fwmr.tableID
	rule.Mark = uint32(fwmr.fwmark)
	rule.Family = fwmr.family()
	err := netlink.RuleAdd(rule)
	if err != nil {
		return fmt.Errorf("%w:%w",
			&FWMarkRouteRuleError{msg: fmt.Sprintf("failed adding rule (%s) while adding fwmark route", rule.String())},
			err)
	}

	// Old ARP/NDP entry for that IP address could cause temporary issue.
	fwmr.cleanNeighbor()

	fwmr.route = &netlink.Route{
		Gw:    fwmr.ip.IP,
		Table: fwmr.tableID,
	}
	err = netlink.RouteAdd(fwmr.route)
	if err != nil {
		return fmt.Errorf("%w: %w",
			&FWMarkRouteError{msg: fmt.Sprintf("failed adding route (%s) while adding fwmark route", fwmr.route.String())},
			err)
	}

	return nil
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
		fwmr.logger.V(1).Info("fetching Neighbor list", "error", err)
		return
	}
	for _, n := range neighbors {
		if n.IP.Equal(fwmr.ip.IP) {
			err = netlink.NeighDel(&n)
			if err != nil {
				fwmr.logger.V(1).Info("delete from neighbor list", "neighbor", n, "error", err)
			}
		}
	}
}

// NewFWMarkRoute -
func NewFWMarkRoute(ip string, fwmark int, tableID int) (*FWMarkRoute, error) {
	netlinkAddr, err := netlink.ParseAddr(ip)
	if err != nil {
		return nil, fmt.Errorf("failed parsing addr (%s) while create fwmark route: %w", ip, err)
	}
	fwMarkRoute := &FWMarkRoute{
		ip:      netlinkAddr,
		fwmark:  fwmark,
		tableID: tableID,
		logger: log.Logger.WithValues("class", "NewFWMarkRoute",
			"ip", ip,
			"fwmark", fwmark,
			"tableID", tableID,
		),
	}
	err = fwMarkRoute.configure()
	if err != nil {
		returnErr := err
		// TODO: should try to remove both rules and routes...
		delErr := fwMarkRoute.Delete()
		if delErr != nil {
			// should not complain about missing rule or route if failed to add them
			needDelError := true
			// lack of rule is ok if configure failed to add it (route should be mssing as well)
			if asErrFWMarkRule(err) && (isErrNoRule(delErr) || isErrNoRoute(delErr)) {
				needDelError = false
			}
			// lack of route is ok if configure failed to add it
			if asErrFWMarkRoute(err) && isErrNoRoute(delErr) {
				needDelError = false
			}
			if needDelError {
				returnErr = fmt.Errorf("%w: fwmark cleanup: %w", err, delErr)
			}
		}
		return nil, returnErr
	}
	return fwMarkRoute, nil
}
