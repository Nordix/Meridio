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
