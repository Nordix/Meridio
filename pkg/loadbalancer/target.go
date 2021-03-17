package loadbalancer

import "github.com/vishvananda/netlink"

type Target struct {
	identifier int
	ip         *netlink.Addr
}

func (t *Target) GetIdentifier() int {
	return t.identifier
}

func (t *Target) GetIP() *netlink.Addr {
	return t.ip
}

func NewTarget(identifier int, ip *netlink.Addr) *Target {
	target := &Target{
		identifier: identifier,
		ip:         ip,
	}
	return target
}
