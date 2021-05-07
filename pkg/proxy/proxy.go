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
	bridge           networking.Bridge
	sourceBasedRoute networking.SourceBasedRoute
	vip              string
	subnet           string
	ipam             *ipam.Ipam
	mutex            sync.Mutex
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
		// Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			err = p.sourceBasedRoute.AddNexthop(ip)
			if err != nil {
				logrus.Errorf("Proxy: Error adding nexthop: %v", err)
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
		// Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			err = p.sourceBasedRoute.RemoveNexthop(ip)
			if err != nil {
				logrus.Errorf("Proxy: Error removing nexthop: %v", err)
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

	srcIPAddr, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return err
	}

	dstIPAddr, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return err
	}
	conn.GetContext().IpContext = &networkservice.IPContext{}
	if interfaceType == networking.NSE {
		conn.GetContext().IpContext.SrcIpAddrs = []string{srcIPAddr}
		conn.GetContext().IpContext.DstIpAddrs = []string{dstIPAddr}
		conn.GetContext().GetIpContext().ExtraPrefixes = p.bridge.GetLocalPrefixes()
	} else if interfaceType == networking.NSC {
		conn.GetContext().IpContext.SrcIpAddrs = []string{dstIPAddr}
		conn.GetContext().IpContext.DstIpAddrs = []string{srcIPAddr}
	}
	return nil
}

func (p *Proxy) setBridgeIP() error {
	prefix, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return err
	}
	err = p.bridge.AddLocalPrefix(prefix)
	if err != nil {
		return err
	}
	p.bridge.SetLocalPrefixes(append(p.bridge.GetLocalPrefixes(), prefix))
	return nil
}

// NewProxy -
func NewProxy(vip string, subnet string, netUtils networking.Utils) *Proxy {
	bridge, err := netUtils.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	sourceBasedRoute, err := netUtils.NewSourceBasedRoute(10, vip)
	if err != nil {
		logrus.Errorf("Proxy: Error creating sourceBasedRoute: %v", err)
	}
	ipam := ipam.NewIpam()
	proxy := &Proxy{
		bridge:           bridge,
		sourceBasedRoute: sourceBasedRoute,
		vip:              vip,
		subnet:           subnet,
		ipam:             ipam,
	}
	err = proxy.setBridgeIP()
	if err != nil {
		logrus.Errorf("Proxy: Error setting the bridge IP: %v", err)
	}
	return proxy
}
