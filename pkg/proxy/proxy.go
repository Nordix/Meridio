package proxy

import (
	"fmt"
	"strconv"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type Proxy struct {
	bridge        *networking.Bridge
	outgoingRoute *networking.OutgoingRoute
	vip           *netlink.Addr
	subnet        *netlink.Addr
	ipam          *ipam.Ipam
}

func (p *Proxy) isNSMInterface(intf *networking.Interface) bool {
	return intf.InteraceType == networking.NSE || intf.InteraceType == networking.NSC
}

func (p *Proxy) InterfaceCreated(intf *networking.Interface) {
	if p.isNSMInterface(intf) == false {
		return
	}
	logrus.Infof("Proxy: interface created: %v", intf)
	// Link the interface to the bridge
	p.bridge.LinkInterface(intf)
	if intf.InteraceType == networking.NSC {
		// Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.NeighborIPs {
			err := p.outgoingRoute.AddNexthop(ip)
			if err != nil {
				logrus.Errorf("Proxy: Error adding nexthop: %v", err)
			}
		}
	}
}

func (p *Proxy) InterfaceDeleted(intf *networking.Interface) {
	if p.isNSMInterface(intf) == false {
		return
	}
	logrus.Infof("Proxy: interface removed: %v", intf)
	// Unlink the interface from the bridge
	p.bridge.UnLinkInterface(intf)
	if intf.InteraceType == networking.NSC {
		// Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.NeighborIPs {
			err := p.outgoingRoute.RemoveNexthop(ip)
			if err != nil {
				logrus.Errorf("Proxy: Error removing nexthop: %v", err)
			}
		}
	}
}

func (p *Proxy) NewNSCIPContext() (*networkservice.IPContext, error) {
	prefixLength, _ := p.subnet.Mask.Size()

	ip, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	dstIpAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))

	ip, err = p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	srcIpAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))

	ipContext := &networkservice.IPContext{
		// DstIpAddr: dstIpAddr, // IP on the NSE
		DstIpAddr: dstIpAddr, // IP on the NSE
		SrcIpAddr: srcIpAddr, // IP on the target
	}
	logrus.Infof("Proxy: koukou NSC: %v", ipContext)
	return ipContext, nil
}

func (p *Proxy) NewNSEIPContext() (*networkservice.IPContext, error) {
	prefixLength, _ := p.subnet.Mask.Size()

	ip, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	srcIpAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))

	ip, err = p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	dstIpAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))

	ipContext := &networkservice.IPContext{
		// SrcIpAddr: srcIpAddr, // IP on the target
		SrcIpAddr: srcIpAddr, // IP on the target
		DstIpAddr: dstIpAddr, // IP on the NSE
	}
	logrus.Infof("Proxy: koukou NSE: %v", ipContext)
	return ipContext, nil
}

func (p *Proxy) setBridgeIP() error {
	ip, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return err
	}
	err = p.bridge.AddAddress(ip)
	if err != nil {
		return err
	}
	return nil
}

func NewProxy(vip *netlink.Addr, subnet *netlink.Addr) *Proxy {
	bridge, err := networking.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	outgoingRoute := networking.NewOutgoingRoute(10, vip)
	ipam := ipam.NewIpam()
	proxy := &Proxy{
		bridge:        bridge,
		outgoingRoute: outgoingRoute,
		vip:           vip,
		subnet:        subnet,
		ipam:          ipam,
	}
	err = proxy.setBridgeIP()
	if err != nil {
		logrus.Errorf("Proxy: Error the bridge IP: %v", err)
	}
	return proxy
}
