/*
Copyright (c) 2021 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
)

const dstChildNamePrefix = "-dst"
const srcChildNamePrefix = "-src"

// Proxy -
type Proxy struct {
	Bridge              networking.Bridge
	vips                []*virtualIP
	conduit             *nspAPI.Conduit
	Subnets             map[ipamAPI.IPFamily]*ipamAPI.Subnet
	IpamClient          ipamAPI.IpamClient
	mutex               sync.Mutex
	netUtils            networking.Utils
	nexthops            []string
	tableID             int
	logger              logr.Logger
	connectionToRelease chan string
}

func (p *Proxy) isNSMInterface(intf networking.Iface) bool {
	return intf.GetInterfaceType() == networking.NSE || intf.GetInterfaceType() == networking.NSC
}

// InterfaceCreated -
func (p *Proxy) InterfaceCreated(intf networking.Iface) {
	if !p.isNSMInterface(intf) {
		return
	}
	p.logger.Info("InterfaceCreated", "intf", intf, "nexthops", p.nexthops)
	// Link the interface to the bridge
	err := p.Bridge.LinkInterface(intf)
	if err != nil {
		p.logger.Error(err, "LinkInterface")
	}
	if intf.GetInterfaceType() == networking.NSC {
		// 	Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			for _, vip := range p.vips {
				err = vip.AddNexthop(ip)
				if err != nil {
					p.logger.Error(err, "Adding nexthop")
				}
			}
			// append nexthop if not known
			add := true
			for _, nexthop := range p.nexthops {
				if nexthop == ip {
					add = false
					break
				}
			}
			if add {
				p.nexthops = append(p.nexthops, ip)
			}
		}
	}
}

// InterfaceDeleted -
func (p *Proxy) InterfaceDeleted(intf networking.Iface) {
	if !p.isNSMInterface(intf) {
		return
	}
	p.logger.Info("InterfaceDeleted", "intf", intf, "nexthops", p.nexthops)
	// Unlink the interface from the bridge
	err := p.Bridge.UnLinkInterface(intf)
	if err != nil {
		p.logger.Error(err, "UnLinkInterface")
	}
	if intf.GetInterfaceType() == networking.NSC {
		// 	Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			for _, vip := range p.vips {
				err = vip.RemoveNexthop(ip)
				if err != nil {
					p.logger.Error(err, "Removing nexthop")
				}
			}

			nexthops := p.nexthops[:0]
			for _, nexthop := range p.nexthops {
				if nexthop != ip {
					nexthops = append(nexthops, nexthop)
				}
			}
			p.nexthops = nexthops
		}
	}
}

// SetIPContext
func (p *Proxy) SetIPContext(ctx context.Context, conn *networkservice.Connection, interfaceType networking.InterfaceType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if conn == nil {
		return errors.New("connection is nil")
	}

	if conn.GetContext() == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.GetContext().GetIpContext() == nil {
		conn.GetContext().IpContext = &networkservice.IPContext{}
	}

	id := conn.Id
	if interfaceType == networking.NSE {
		// For TAPA originated connections (i.e. when Proxy acts as NSE) use the
		// segment ID referring to the TAPA side of the NSM connection from now on.
		// The ID is used as key by IPAM to store/lookup the associated IP addresses.
		// The intial segment of the NSM connection represent the TAPA (NSC), thus it
		// will stay intact even if the Proxy NSM segment change (e.g. after POD kill or upgrade).
		//
		// This is an NBC. (It could be addrssed by introducing versioning for the NSM connection or by
		// adding excess code to try and recover IPs the old way as well.)
		id = conn.GetPath().GetPathSegments()[0].Id
	}

	srcIPAddrs := []string{}
	dstIpAddrs := []string{}
	for _, subnet := range p.Subnets {
		child := &ipamAPI.Child{
			Name:   fmt.Sprintf("%s%s", id, srcChildNamePrefix),
			Subnet: subnet,
		}
		srcPrefix, err := p.IpamClient.Allocate(ctx, child)
		if err != nil {
			return err
		}
		srcIPAddrs = append(srcIPAddrs, srcPrefix.ToString())

		child = &ipamAPI.Child{
			Name:   fmt.Sprintf("%s%s", id, dstChildNamePrefix),
			Subnet: subnet,
		}
		dstPrefix, err := p.IpamClient.Allocate(ctx, child)
		if err != nil {
			return err
		}
		dstIpAddrs = append(dstIpAddrs, dstPrefix.ToString())
	}

	if interfaceType == networking.NSE {
		p.setNSEIpContext(conn.GetContext().GetIpContext(), srcIPAddrs, dstIpAddrs)
	} else if interfaceType == networking.NSC {
		conn.GetContext().GetIpContext().SrcIpAddrs = dstIpAddrs
		conn.GetContext().GetIpContext().DstIpAddrs = srcIPAddrs
	}
	return nil
}

func (p *Proxy) setNSEIpContext(ipContext *networkservice.IPContext, srcIPAddrs []string, dstIpAddrs []string) {
	if len(ipContext.SrcIpAddrs) == 0 && len(ipContext.DstIpAddrs) == 0 { // First request
		ipContext.SrcIpAddrs = srcIPAddrs
		ipContext.DstIpAddrs = dstIpAddrs
		ipContext.ExtraPrefixes = p.Bridge.GetLocalPrefixes()
		return
	}
	// The request is an update
	if contains(ipContext.ExtraPrefixes, p.Bridge.GetLocalPrefixes()) &&
		contains(ipContext.SrcIpAddrs, srcIPAddrs) &&
		contains(ipContext.DstIpAddrs, dstIpAddrs) {
		return
	}
	// remove old IPs, add new ones, and set the gateways
	oldGateways := ipContext.GetExtraPrefixes()
	ipContext.ExtraPrefixes = p.Bridge.GetLocalPrefixes()
	ipContext.SrcIpAddrs = removeOldIPs(ipContext.SrcIpAddrs, oldGateways)
	ipContext.DstIpAddrs = removeOldIPs(ipContext.DstIpAddrs, oldGateways)
	ipContext.SrcIpAddrs = append(ipContext.SrcIpAddrs, srcIPAddrs...)
	ipContext.DstIpAddrs = append(ipContext.DstIpAddrs, dstIpAddrs...)
	// Find IPv4 gateway and IPv6 Gateway
	gatewayV4 := ""
	gatewayV6 := ""
	for _, gw := range ipContext.ExtraPrefixes {
		ip, _, err := net.ParseCIDR(gw)
		if err != nil {
			continue
		}
		if ip.To4() != nil {
			gatewayV4 = ip.String()
			continue
		}
		gatewayV6 = ip.String()
	}
	// Replace the nexthops in the policy routes with the up to date gateways
	for _, policyRoute := range ipContext.GetPolicies() {
		ipv4 := true
		ip, _, err := net.ParseCIDR(policyRoute.From)
		if err != nil {
			continue
		}
		if ip.To4() == nil {
			ipv4 = false
		}
		for _, route := range policyRoute.Routes {
			if ipv4 {
				route.NextHop = gatewayV4
			} else {
				route.NextHop = gatewayV6
			}
		}
	}
}

// removes all IPs in the ips list that are in the same subnet as any of the gateway
func removeOldIPs(ips []string, gateways []string) []string {
	gws := []*net.IPNet{}
	for _, ip := range gateways {
		_, n, err := net.ParseCIDR(ip)
		if err != nil {
			continue
		}
		gws = append(gws, n)
	}
	res := []string{}
	for _, ip := range ips {
		i, _, err := net.ParseCIDR(ip)
		if err != nil {
			continue
		}
		ipInGatewaySubnet := false
		for _, net := range gws {
			if net.Contains(i) {
				ipInGatewaySubnet = true
			}
		}
		if !ipInGatewaySubnet {
			res = append(res, ip)
		}
	}
	return res
}

// Tells if a contains all b items
func contains(a []string, b []string) bool {
	aMap := listToMap(a)
	for _, i := range b {
		_, exists := aMap[i]
		if !exists {
			return false
		}
	}
	return true
}

// convert a list of string to a map with values as key
func listToMap(l []string) map[string]struct{} {
	res := map[string]struct{}{}
	for _, s := range l {
		res[s] = struct{}{}
	}
	return res
}

func (p *Proxy) UnsetIPContext(ctx context.Context, conn *networkservice.Connection, interfaceType networking.InterfaceType) error {
	id := conn.Id
	if interfaceType == networking.NSE {
		// Use the segment ID referring to the TAPA side of the NSM connection
		id = conn.GetPath().GetPathSegments()[0].Id
	}
	p.connectionToRelease <- id
	return nil
}

func (p *Proxy) IPReleaser(ctx context.Context) {
	for {
		select {
		case id := <-p.connectionToRelease:
			ctxRelease, cancel := context.WithTimeout(ctx, 10*time.Second)
			for _, subnet := range p.Subnets {
				child := &ipamAPI.Child{
					Name:   fmt.Sprintf("%s%s", id, srcChildNamePrefix),
					Subnet: subnet,
				}
				_, err := p.IpamClient.Release(ctxRelease, child)
				if err != nil {
					p.logger.Error(err, "failed to release IP", "id", id, "subnet", subnet)
					select {
					case p.connectionToRelease <- id:
						p.connectionToRelease <- id
					default:
						p.logger.V(1).Info("dropping IP release since connectionToRelease if full", "id", id, "subnet", subnet)
					}
					break
				}
				child = &ipamAPI.Child{
					Name:   fmt.Sprintf("%s%s", id, dstChildNamePrefix),
					Subnet: subnet,
				}
				_, err = p.IpamClient.Release(ctxRelease, child)
				if err != nil {
					p.logger.Error(err, "failed to release IP", "id", id, "subnet", subnet)
					select {
					case p.connectionToRelease <- id:
						p.connectionToRelease <- id
					default:
						p.logger.V(1).Info("dropping IP release since connectionToRelease if full", "id", id, "subnet", subnet)
					}
					break
				}
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func (p *Proxy) setBridgeIP(prefix string) error {
	err := p.Bridge.AddLocalPrefix(prefix)
	if err != nil {
		return err
	}
	p.Bridge.SetLocalPrefixes(append(p.Bridge.GetLocalPrefixes(), prefix))
	return nil
}

func (p *Proxy) setBridgeIPs() error {
	for _, subnet := range p.Subnets {
		child := &ipamAPI.Child{
			Name:   "bridge",
			Subnet: subnet,
		}
		prefix, err := p.IpamClient.Allocate(context.TODO(), child)
		if err != nil {
			return err
		}
		err = p.setBridgeIP(prefix.ToString())
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
				p.logger.Error(err, "Adding SourceBaseRoute")
				continue
			}
			p.tableID++
			p.vips = append(p.vips, newVIP)
			for _, nexthop := range p.nexthops {
				err = newVIP.AddNexthop(nexthop)
				if err != nil {
					p.logger.Error(err, "Adding nexthop")
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
				p.logger.Error(err, "Deleting vip")
			}
		}
	}
}

// NewProxy -
func NewProxy(conduit *nspAPI.Conduit, nodeName string, ipamClient ipamAPI.IpamClient, ipFamily string, netUtils networking.Utils) *Proxy {
	logger := log.Logger.WithValues("class", "Proxy")
	bridge, err := netUtils.NewBridge("bridge0")
	if err != nil {
		logger.Error(err, "Creating the bridge")
	}
	proxy := &Proxy{
		Bridge:              bridge,
		conduit:             conduit,
		netUtils:            netUtils,
		nexthops:            []string{},
		vips:                []*virtualIP{},
		tableID:             1,
		Subnets:             make(map[ipamAPI.IPFamily]*ipamAPI.Subnet),
		IpamClient:          ipamClient,
		logger:              logger,
		connectionToRelease: make(chan string, 40),
	}

	if strings.ToLower(ipFamily) == "ipv4" {
		proxy.Subnets[ipamAPI.IPFamily_IPV4] = &ipamAPI.Subnet{
			Conduit:  conduit,
			Node:     nodeName,
			IpFamily: ipamAPI.IPFamily_IPV4,
		}
	} else if strings.ToLower(ipFamily) == "ipv6" {
		proxy.Subnets[ipamAPI.IPFamily_IPV6] = &ipamAPI.Subnet{
			Conduit:  conduit,
			Node:     nodeName,
			IpFamily: ipamAPI.IPFamily_IPV6,
		}
	} else {
		proxy.Subnets[ipamAPI.IPFamily_IPV4] = &ipamAPI.Subnet{
			Conduit:  conduit,
			Node:     nodeName,
			IpFamily: ipamAPI.IPFamily_IPV4,
		}
		proxy.Subnets[ipamAPI.IPFamily_IPV6] = &ipamAPI.Subnet{
			Conduit:  conduit,
			Node:     nodeName,
			IpFamily: ipamAPI.IPFamily_IPV6,
		}
	}
	err = proxy.setBridgeIPs()
	if err != nil {
		logger.Error(err, "Setting the bridge IP")
	}
	return proxy
}
