package loadbalancer

import "github.com/vishvananda/netlink"

type Target struct {
	identifier int
	ip         *netlink.Addr
}

func NewTarget(identifier int, ip *netlink.Addr) *Target {
	target := &Target{
		identifier: identifier,
		ip:         ip,
	}
	return target
}
