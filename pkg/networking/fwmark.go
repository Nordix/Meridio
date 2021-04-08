package networking

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
	return netlink.RuleDel(rule)
}

func (fwmr *FWMarkRoute) configure() error {
	rule := netlink.NewRule()
	rule.Table = fwmr.tableID
	rule.Mark = fwmr.fwmark
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

// NewFWMarkRoute -
func NewFWMarkRoute(ip *netlink.Addr, fwmark int, tableID int) (*FWMarkRoute, error) {
	fwMarkRoute := &FWMarkRoute{
		ip:      ip,
		fwmark:  fwmark,
		tableID: tableID,
	}
	err := fwMarkRoute.configure()
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
