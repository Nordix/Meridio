package networking

import "github.com/vishvananda/netlink"

func AddVIP(vip *netlink.Addr) error {
	loopbackInterface, err := netlink.LinkByName("lo")
	if err != nil {
		return err
	}
	err = netlink.AddrAdd(loopbackInterface, vip)
	if err != nil {
		return err
	}
	return nil
}
