package loadbalancer

import "github.com/vishvananda/netlink"

type Target struct {
	identifier int
	ip         *netlink.Addr
}
