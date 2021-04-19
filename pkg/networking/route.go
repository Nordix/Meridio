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

func (sbr *SourceBasedRoute) create() error {
	rule := netlink.NewRule()
	rule.Table = sbr.tableID
	rule.Src = &net.IPNet{
		IP:   sbr.vip.IP,
		Mask: sbr.vip.Mask,
	}
	rule.Family = sbr.family()
	return netlink.RuleAdd(rule)
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
	return netlink.RouteReplace(route)
}

// AddNexthop -
func (sbr *SourceBasedRoute) AddNexthop(nexthop *netlink.Addr) error {
	sbr.nexthops = append(sbr.nexthops, nexthop)
	return sbr.updateRoute()
}

// RemoveNexthop -
func (sbr *SourceBasedRoute) RemoveNexthop(nexthop *netlink.Addr) error {
	for index, current := range sbr.nexthops {
		if nexthop.IP.String() == current.IP.String() {
			sbr.nexthops = append(sbr.nexthops[:index], sbr.nexthops[index+1:]...)
		}
	}
	return sbr.updateRoute()
}

func (sbr *SourceBasedRoute) family() int {
	if sbr.vip.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
}

// NewSourceBasedRoute -
func NewSourceBasedRoute(tableID int, vip *netlink.Addr) (*SourceBasedRoute, error) {
	sourceBasedRoute := &SourceBasedRoute{
		tableID:  tableID,
		vip:      vip,
		nexthops: []*netlink.Addr{},
	}
	err := sourceBasedRoute.create()
	if err != nil {
		return nil, err
	}
	return sourceBasedRoute, nil
}
