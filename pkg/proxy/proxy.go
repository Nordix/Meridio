package proxy

import (
	"github.com/nordix/nvip/pkg/networking"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	bridge        *networking.Bridge
	outgoingRoute *networking.OutgoingRoute
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
		p.bridge.AddAddress(ip)
		intf.RemoveAddress(ip)
	}
	if intf.InteraceType == networking.NSC {
		// Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.NeighborIPs {
			p.outgoingRoute.AddNexthop(ip)
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
		p.bridge.RemoveAddress(ip)
		intf.AddAddress(ip)
	}
	if intf.InteraceType == networking.NSC {
		// Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.NeighborIPs {
			p.outgoingRoute.RemoveNexthop(ip)
		}
	}
}

func NewProxy() *Proxy {
	bridge, err := networking.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	proxy := &Proxy{
		bridge: bridge,
	}
	return proxy
}
