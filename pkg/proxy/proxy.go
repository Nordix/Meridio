package proxy

import (
	"errors"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

// Proxy -
type Proxy struct {
	bridge            networking.Bridge
	sourceBasedRoutes []networking.SourceBasedRoute
	vips              []string
	subnets           []string
	ipam              *ipam.Ipam
	mutex             sync.Mutex
	netUtils          networking.Utils
}

func (p *Proxy) isNSMInterface(intf networking.Iface) bool {
	return intf.GetInterfaceType() == networking.NSE || intf.GetInterfaceType() == networking.NSC
}

// InterfaceCreated -
func (p *Proxy) InterfaceCreated(intf networking.Iface) {
	if !p.isNSMInterface(intf) {
		return
	}
	logrus.Infof("Proxy: interface created: %v", intf)
	// Link the interface to the bridge
	err := p.bridge.LinkInterface(intf)
	if err != nil {
		logrus.Errorf("Proxy: Error LinkInterface: %v", err)
	}
	if intf.GetInterfaceType() == networking.NSC {
		// 	Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			for _, sourceBasedRoute := range p.sourceBasedRoutes {
				err = sourceBasedRoute.AddNexthop(ip)
				if err != nil {
					logrus.Errorf("Proxy: Error adding nexthop: %v", err)
				}
			}
		}
	}
}

// InterfaceDeleted -
func (p *Proxy) InterfaceDeleted(intf networking.Iface) {
	if !p.isNSMInterface(intf) {
		return
	}
	logrus.Infof("Proxy: interface removed: %v", intf)
	// Unlink the interface from the bridge
	err := p.bridge.UnLinkInterface(intf)
	if err != nil {
		logrus.Errorf("Proxy: Error UnLinkInterface: %v", err)
	}
	if intf.GetInterfaceType() == networking.NSC {
		// 	Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			for _, sourceBasedRoute := range p.sourceBasedRoutes {
				err = sourceBasedRoute.RemoveNexthop(ip)
				if err != nil {
					logrus.Errorf("Proxy: Error removing nexthop: %v", err)
				}
			}
		}
	}
}

// SetIPContext
func (p *Proxy) SetIPContext(conn *networkservice.Connection, interfaceType networking.InterfaceType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if conn == nil {
		return errors.New("connection is nil")
	}

	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}

	srcIPAddrs := []string{}
	dstIpAddrs := []string{}
	for _, subnet := range p.subnets {
		srcIPAddr, err := p.ipam.AllocateIP(subnet)
		if err != nil {
			return err
		}
		srcIPAddrs = append(srcIPAddrs, srcIPAddr)

		dstIPAddr, err := p.ipam.AllocateIP(subnet)
		if err != nil {
			return err
		}
		dstIpAddrs = append(dstIpAddrs, dstIPAddr)
	}

	conn.GetContext().IpContext = &networkservice.IPContext{}
	if interfaceType == networking.NSE {
		conn.GetContext().IpContext.SrcIpAddrs = srcIPAddrs
		conn.GetContext().IpContext.DstIpAddrs = dstIpAddrs
		conn.GetContext().GetIpContext().ExtraPrefixes = p.bridge.GetLocalPrefixes()
	} else if interfaceType == networking.NSC {
		conn.GetContext().IpContext.SrcIpAddrs = dstIpAddrs
		conn.GetContext().IpContext.DstIpAddrs = srcIPAddrs
	}
	return nil
}

func (p *Proxy) setBridgeIP(prefix string) error {
	err := p.bridge.AddLocalPrefix(prefix)
	if err != nil {
		return err
	}
	p.bridge.SetLocalPrefixes(append(p.bridge.GetLocalPrefixes(), prefix))
	return nil
}

func (p *Proxy) setBridgeIPs() error {
	for _, subnet := range p.subnets {
		prefix, err := p.ipam.AllocateIP(subnet)
		if err != nil {
			return err
		}
		err = p.setBridgeIP(prefix)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) createSourceBaseRoutes() error {
	for index, vip := range p.vips {
		sourceBasedRoute, err := p.netUtils.NewSourceBasedRoute(index, vip)
		logrus.Infof("Proxy: sourceBasedRoute index - vip: %v - %v", index, vip)
		if err != nil {
			return err
		}
		p.sourceBasedRoutes = append(p.sourceBasedRoutes, sourceBasedRoute)
	}
	return nil
}

// NewProxy -
func NewProxy(vips []string, subnets []string, netUtils networking.Utils) *Proxy {
	bridge, err := netUtils.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	ipam := ipam.NewIpam()
	proxy := &Proxy{
		bridge:            bridge,
		sourceBasedRoutes: []networking.SourceBasedRoute{},
		vips:              vips,
		subnets:           subnets,
		ipam:              ipam,
		netUtils:          netUtils,
	}
	err = proxy.setBridgeIPs()
	if err != nil {
		logrus.Errorf("Proxy: Error setting the bridge IP: %v", err)
	}
	err = proxy.createSourceBaseRoutes()
	if err != nil {
		logrus.Errorf("Proxy: Error createSourceBaseRoutes: %v", err)
	}
	return proxy
}
