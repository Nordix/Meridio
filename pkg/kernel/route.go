package kernel

import (
	"net"
	"sync"

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
		return err
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
		return err
	}
	sbr.removeFromList(netlinkAddr)
	err = sbr.updateRoute()
	return err
}

func (sbr *SourceBasedRoute) Delete() error {
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
		return err
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
	return netlink.RouteDel(route)
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
		return nil, err
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
