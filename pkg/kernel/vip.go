package kernel

import "github.com/vishvananda/netlink"

func AddVIP(vip string) error {
	netlinkAddr, err := netlink.ParseAddr(vip)
	if err != nil {
		return err
	}
	loopbackInterface, err := netlink.LinkByName("lo")
	if err != nil {
		return err
	}
	err = netlink.AddrAdd(loopbackInterface, netlinkAddr)
	if err != nil {
		return err
	}
	return nil
}

func DeleteVIP(vip string) error {
	netlinkAddr, err := netlink.ParseAddr(vip)
	if err != nil {
		return err
	}
	loopbackInterface, err := netlink.LinkByName("lo")
	if err != nil {
		return err
	}
	err = netlink.AddrDel(loopbackInterface, netlinkAddr)
	if err != nil {
		return err
	}
	return nil
}
