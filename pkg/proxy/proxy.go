package proxy

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/ipam"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Proxy -
type Proxy struct {
	bridge           *networking.Bridge
	sourceBasedRoute *networking.SourceBasedRoute
	vip              *netlink.Addr
	subnet           *netlink.Addr
	ipam             *ipam.Ipam
	mutex            sync.Mutex
}

func (p *Proxy) isNSMInterface(intf *networking.Interface) bool {
	return intf.InteraceType == networking.NSE || intf.InteraceType == networking.NSC
}

// InterfaceCreated -
func (p *Proxy) InterfaceCreated(intf *networking.Interface) {
	if !p.isNSMInterface(intf) {
		return
	}
	logrus.Infof("Proxy: interface created: %v", intf)
	// Link the interface to the bridge
	err := p.bridge.LinkInterface(intf)
	if err != nil {
		logrus.Errorf("Proxy: Error LinkInterface: %v", err)
	}
	if intf.InteraceType == networking.NSC {
		// Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.NeighborIPs {
			err = p.sourceBasedRoute.AddNexthop(ip)
			if err != nil {
				logrus.Errorf("Proxy: Error adding nexthop: %v", err)
			}
		}
	}
}

// InterfaceDeleted -
func (p *Proxy) InterfaceDeleted(intf *networking.Interface) {
	if !p.isNSMInterface(intf) {
		return
	}
	logrus.Infof("Proxy: interface removed: %v", intf)
	// Unlink the interface from the bridge
	err := p.bridge.UnLinkInterface(intf)
	if err != nil {
		logrus.Errorf("Proxy: Error UnLinkInterface: %v", err)
	}
	if intf.InteraceType == networking.NSC {
		// Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.NeighborIPs {
			err = p.sourceBasedRoute.RemoveNexthop(ip)
			if err != nil {
				logrus.Errorf("Proxy: Error removing nexthop: %v", err)
			}
		}
	}
}

// NewNSCIPContext -
func (p *Proxy) NewNSCIPContext() (*networkservice.IPContext, error) {
	prefixLength, _ := p.subnet.Mask.Size()

	p.mutex.Lock()
	ip, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	dstIPAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))

	ip, err = p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	srcIPAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))
	p.mutex.Unlock()

	ipContext := &networkservice.IPContext{
		// DstIpAddr: dstIpAddr, // IP on the NSE
		DstIpAddr: dstIPAddr, // IP on the NSE
		SrcIpAddr: srcIPAddr, // IP on the target
	}
	return ipContext, nil
}

// NewNSEIPContext -
func (p *Proxy) NewNSEIPContext() (*networkservice.IPContext, error) {
	prefixLength, _ := p.subnet.Mask.Size()

	p.mutex.Lock()
	ip, err := p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	srcIPAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))

	ip, err = p.ipam.AllocateIP(p.subnet)
	if err != nil {
		return nil, err
	}
	dstIPAddr := fmt.Sprintf("%s/%s", ip.IP.String(), strconv.Itoa(prefixLength))
	p.mutex.Unlock()

	ipContext := &networkservice.IPContext{
		// SrcIpAddr: srcIpAddr, // IP on the target
		SrcIpAddr: srcIPAddr, // IP on the target
		DstIpAddr: dstIPAddr, // IP on the NSE
	}
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

// NewProxy -
func NewProxy(vip *netlink.Addr, subnet *netlink.Addr) *Proxy {
	bridge, err := networking.NewBridge("bridge0")
	if err != nil {
		logrus.Errorf("Proxy: Error creating the bridge: %v", err)
	}
	sourceBasedRoute, err := networking.NewSourceBasedRoute(10, vip)
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
