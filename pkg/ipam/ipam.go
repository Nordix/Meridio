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

func (ipam *Ipam) AllocateSubnet(subnetPool string, prefixLength int) (string, error) {
	_, err := netlink.ParseAddr(subnetPool)
	if err != nil {
		return "", err
	}
	err = ipam.registerPrefix(subnetPool)
	if err != nil {
		return "", err
	}
	child, err := ipam.goIpam.AcquireChildPrefix(subnetPool, uint8(prefixLength))
	if err != nil {
		return "", err
	}
	return child.Cidr, nil
}

func (ipam *Ipam) ReleaseSubnet(subnet string) error {
	_, err := netlink.ParseAddr(subnet)
	if err != nil {
		return err
	}
	err = ipam.registerPrefix(subnet)
	if err != nil {
		return err
	}
	// TODO
	return nil
}

func (ipam *Ipam) AllocateIP(subnet string) (string, error) {
	netlinkSubnet, err := netlink.ParseAddr(subnet)
	if err != nil {
		return "", err
	}
	err = ipam.registerPrefix(subnet)
	if err != nil {
		return "", err
	}
	ip, err := ipam.goIpam.AcquireIP(subnet)
	if err != nil {
		return "", err
	}
	prefixLength, _ := netlinkSubnet.Mask.Size()
	ipCidr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))
	return ipCidr, nil
}

func (ipam *Ipam) ReleaseIP(ip string) error {
	Addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return err
	}
	err = ipam.registerPrefix(Addr.Network())
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
