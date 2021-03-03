package ipam

import (
	"fmt"
	"strconv"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/vishvananda/netlink"
)

type Ipam struct {
	goIpam           goipam.Ipamer
	registeredPrefix map[string]struct{}
}

func (ipam *Ipam) registerPrefix(prefix string) error {
	if _, ok := ipam.registeredPrefix[prefix]; ok {
		return nil
	}
	ipam.registeredPrefix[prefix] = struct{}{}
	_, err := ipam.goIpam.NewPrefix(prefix)
	return err
}

func (ipam *Ipam) AllocateSubnet(subnetPool *netlink.Addr, prefixLength int) (*netlink.Addr, error) {
	err := ipam.registerPrefix(subnetPool.String())
	if err != nil {
		return nil, err
	}
	child, err := ipam.goIpam.AcquireChildPrefix(subnetPool.String(), uint8(prefixLength))
	if err != nil {
		return nil, err
	}
	return netlink.ParseAddr(child.Cidr)
}

func (ipam *Ipam) ReleaseSubnet(subnet *netlink.Addr) error {
	err := ipam.registerPrefix(subnet.String())
	if err != nil {
		return err
	}
	// TODO
	return nil
}

func (ipam *Ipam) AllocateIP(subnet *netlink.Addr) (*netlink.Addr, error) {
	err := ipam.registerPrefix(subnet.String())
	if err != nil {
		return nil, err
	}
	ip, err := ipam.goIpam.AcquireIP(subnet.String())
	if err != nil {
		return nil, err
	}
	prefixLength, _ := subnet.Mask.Size()
	ipCidr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))
	return netlink.ParseAddr(ipCidr)
}

func (ipam *Ipam) ReleaseIP(ip *netlink.Addr) error {
	err := ipam.registerPrefix(ip.Network())
	if err != nil {
		return err
	}
	// TODO
	return nil
}

func NewIpam() *Ipam {
	goIpam := goipam.New()
	ipam := &Ipam{
		goIpam:           goIpam,
		registeredPrefix: make(map[string]struct{}),
	}
	return ipam
}
