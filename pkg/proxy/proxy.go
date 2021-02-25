package proxy

import (
	"github.com/nordix/nvip/pkg/networking"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type Proxy struct {
	bridge        *networking.Bridge
	outgoingRoute *networking.OutgoingRoute
	vip           *netlink.Addr
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
	// Add addresses to the bridge and remove them from the interface
	for _, ip := range intf.LocalIPs {
		err := p.bridge.AddAddress(ip)
		if err != nil {
			logrus.Errorf("Proxy: Error adding IP on the bridge: %v", err)
		}
		err = intf.RemoveAddress(ip)
		if err != nil {
			logrus.Errorf("Proxy: Error remove IP from an interface: %v", err)
		}
	}
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
	// Add addresses to the interface and remove them from the bridge
	for _, ip := range intf.LocalIPs {
		err := p.bridge.RemoveAddress(ip)
		if err != nil {
			logrus.Errorf("Proxy: Error remove IP from the bridge: %v", err)
		}
		// TODO
		err = intf.AddAddress(ip)
		if err != nil {
			logrus.Errorf("Proxy: Error adding IP on an interface: %v", err)
		}
	}
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

func NewProxy(vip *netlink.Addr) *Proxy {
	bridge, err := networking.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	outgoingRoute := networking.NewOutgoingRoute(10, vip)
	proxy := &Proxy{
		bridge:        bridge,
		outgoingRoute: outgoingRoute,
		vip:           vip,
	}
	return proxy
}
