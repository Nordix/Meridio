package networking

import (
	"os/exec"
	"strconv"

	"github.com/vishvananda/netlink"
)

type OutgoingRoute struct {
	tableId  int
	vip      *netlink.Addr
	nexthops []*netlink.Addr
}

func (or *OutgoingRoute) create() error {
	ipRuleCmd := "/usr/sbin/ip rule add from " + or.vip.IP.String() + " table " + strconv.Itoa(or.tableId) + " priority 10"
	cmd := exec.Command("bash", "-c", ipRuleCmd)
	_, err := cmd.Output()
	return err
}

func (or *OutgoingRoute) updateRoute() error {
	routeCmd := "/usr/sbin/ip route replace table " + strconv.Itoa(or.tableId) + " default "
	for _, nexthop := range or.nexthops {
		routeCmd += " nexthop via " + nexthop.IP.String()
	}
	cmd := exec.Command("bash", "-c", routeCmd)
	_, err := cmd.Output()
	return err
}

func (or *OutgoingRoute) AddNexthop(nexthop *netlink.Addr) error {
	or.nexthops = append(or.nexthops, nexthop)
	return or.updateRoute()
}

func (or *OutgoingRoute) RemoveNexthop(nexthop *netlink.Addr) error {
	for index, current := range or.nexthops {
		if nexthop.IP.String() == current.IP.String() {
			or.nexthops = append(or.nexthops[:index], or.nexthops[index+1:]...)
		}
	}
	return or.updateRoute()
}

func NewOutgoingRoute(tableId int, vip *netlink.Addr) *OutgoingRoute {
	outgoingRoute := &OutgoingRoute{
		tableId:  tableId,
		vip:      vip,
		nexthops: []*netlink.Addr{},
	}
	outgoingRoute.create()
	outgoingRoute.updateRoute()
	return outgoingRoute
}
