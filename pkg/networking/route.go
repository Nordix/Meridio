package networking

import (
	"net"

	"github.com/vishvananda/netlink"
)

// SourceBasedRoute -
type SourceBasedRoute struct {
	tableID  int
	vip      *netlink.Addr
	nexthops []*netlink.Addr
}

func (or *SourceBasedRoute) create() error {
	rule := netlink.NewRule()
	rule.Table = or.tableID
	rule.Src = &net.IPNet{
		IP:   or.vip.IP,
		Mask: or.vip.Mask,
	}
	return netlink.RuleAdd(rule)
}

func (or *SourceBasedRoute) updateRoute() error {
	nexthops := []*netlink.NexthopInfo{}
	for _, nexthop := range or.nexthops {
		nexthops = append(nexthops, &netlink.NexthopInfo{
			Gw: nexthop.IP,
		})
	}

	route := &netlink.Route{
		Table:     or.tableID,
		MultiPath: nexthops,
		Src:       net.IPv4(0, 0, 0, 0),
	}
	return netlink.RouteReplace(route)
}

// AddNexthop -
func (or *SourceBasedRoute) AddNexthop(nexthop *netlink.Addr) error {
	or.nexthops = append(or.nexthops, nexthop)
	return or.updateRoute()
}

// RemoveNexthop -
func (or *SourceBasedRoute) RemoveNexthop(nexthop *netlink.Addr) error {
	for index, current := range or.nexthops {
		if nexthop.IP.String() == current.IP.String() {
			or.nexthops = append(or.nexthops[:index], or.nexthops[index+1:]...)
		}
	}
	return or.updateRoute()
}

// NewSourceBasedRoute -
func NewSourceBasedRoute(tableID int, vip *netlink.Addr) *SourceBasedRoute {
	outgoingRoute := &SourceBasedRoute{
		tableID:  tableID,
		vip:      vip,
		nexthops: []*netlink.Addr{},
	}
	outgoingRoute.create()
	outgoingRoute.updateRoute()
	return outgoingRoute
}
