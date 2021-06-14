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
	bridge   networking.Bridge
	vips     []*virtualIP
	subnets  []string
	ipam     *ipam.Ipam
	mutex    sync.Mutex
	netUtils networking.Utils
	nexthops []string
	tableID  int
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
			for _, vip := range p.vips {
				err = vip.AddNexthop(ip)
				if err != nil {
					logrus.Errorf("Proxy: Error adding nexthop: %v", err)
				}
			}
			p.nexthops = append(p.nexthops, ip)
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
			for _, vip := range p.vips {
				err = vip.RemoveNexthop(ip)
				if err != nil {
					logrus.Errorf("Proxy: Error removing nexthop: %v", err)
				}
			}
			for index, nexthop := range p.nexthops {
				if nexthop == ip {
					p.nexthops = append(p.nexthops[:index], p.nexthops[index+1:]...)
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

func (p *Proxy) SetVIPs(vips []string) {
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range p.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			newVIP, err := newVirtualIP(vip, p.tableID, p.netUtils)
			if err != nil {
				logrus.Errorf("Proxy: Error adding SourceBaseRoute: %v", err)
				continue
			}
			p.tableID++
			p.vips = append(p.vips, newVIP)
			for _, nexthop := range p.nexthops {
				err = newVIP.AddNexthop(nexthop)
				if err != nil {
					logrus.Errorf("Proxy: Error adding nexthop: %v", err)
				}
			}
		}
		delete(currentVIPs, vip)
	}
	// delete remaining vips
	for index := 0; index < len(p.vips); index++ {
		vip := p.vips[index]
		if _, ok := currentVIPs[vip.prefix]; ok {
			p.vips = append(p.vips[:index], p.vips[index+1:]...)
			index--
			err := vip.Delete()
			if err != nil {
				logrus.Errorf("Proxy: Error deleting vip: %v", err)
			}
		}
	}
}

// NewProxy -
func NewProxy(vips []string, subnets []string, netUtils networking.Utils) *Proxy {
	bridge, err := netUtils.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	ipam := ipam.NewIpam()
	proxy := &Proxy{
		bridge:   bridge,
		subnets:  subnets,
		ipam:     ipam,
		netUtils: netUtils,
		nexthops: []string{},
		vips:     []*virtualIP{},
		tableID:  1,
	}
	err = proxy.setBridgeIPs()
	if err != nil {
		logrus.Errorf("Proxy: Error setting the bridge IP: %v", err)
	}
	proxy.SetVIPs(vips)
	return proxy
}
