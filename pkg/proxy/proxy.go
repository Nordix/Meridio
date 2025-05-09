/*
Copyright (c) 2021-2023 Nordix Foundation
Copyright (c) 2024-2025 OpenInfra Foundation Europe

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
	"io/fs"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/nordix/meridio/pkg/utils"
	"github.com/vishvananda/netlink"
)

const (
	dstChildNamePrefix     = "-dst"
	srcChildNamePrefix     = "-src"
	tableID            int = 1 // routing table ID is shared both for installing nexthop routes and for setting up VIP src IP rules as part of source-based routing
)

var once sync.Once

// Proxy -
type Proxy struct {
	Bridge                   networking.Bridge
	vips                     []*virtualIP
	conduit                  *nspAPI.Conduit
	Subnets                  map[ipamAPI.IPFamily]*ipamAPI.Subnet
	IpamClient               ipamAPI.IpamClient
	mutex                    sync.Mutex
	netUtils                 networking.Utils
	nexthops                 map[string]struct{}
	nexthopRoute             *kernel.NexthopRoute
	logger                   logr.Logger
	connectionToReleaseMap   map[string]context.CancelFunc
	connectionToReleaseMutex sync.Mutex
	mu                       sync.Mutex
}

func (p *Proxy) isNSMInterface(intf networking.Iface) bool {
	return intf.GetInterfaceType() == networking.NSE || intf.GetInterfaceType() == networking.NSC
}

// InterfaceCreated -
//
// Note: A NSM connection for which there's already a networking.Iface linked to
// the bridge can be updated on the fly. It might involve update of the src/dst
// addresses for example, which might confuse the bridge if the intefaces would
// be always compared by considering all the networking.Iface fields. Therefore,
// it is expected that the bridge can handle such cases for example by doing a
// lookup based on interface index as well. (We expect only one networking.Iface
// per interface index at a time. Which means, the proxy must handle interface
// delete events originating from the kernel to reconfigure linked interfaces.)
//
// Note: Seemingly, upon NSM connncetion refresh the src and dst addresses might
// appear in a different order then before.
//
// Note: Watch out when using intf.GetName(), because by default it will load
// the interface name into the underlying struct. Which can break the logic
// built on top of comparing interfaces as it relies on "DeepEqual".
func (p *Proxy) InterfaceCreated(intf networking.Iface) {
	if !p.isNSMInterface(intf) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// TODO: Check why interface is not created with a name in the first place?
	// TODO: Consider reworking the whole interface event handling part, including
	// what gets stored and how is handled.
	if !p.Bridge.InterfaceIsLinked(intf) { // avoid NSM connection refresh triggered spamming
		p.logger.Info("InterfaceCreated", "name", intf.GetName(networking.WithNoLoad()),
			"index", intf.GetIndex(), "intf", intf, "nexthops", p.nexthops)
	}
	// Link the interface to the bridge (if already linked update interface fields in case of address changes)
	err := p.Bridge.LinkInterface(intf)
	if err != nil {
		p.logger.Error(err, "LinkInterface")
	}
	if intf.GetInterfaceType() == networking.NSC {
		// 	Add the neighbor IPs of the interface to the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			err = p.nexthopRoute.AddNexthop(ip)
			if err != nil && !errors.Is(err, fs.ErrExist) {
				p.logger.Error(err, "Adding nexthop")
			}
			// append nexthop if not known
			p.nexthops[ip] = struct{}{}
		}
	}
}

// InterfaceDeleted -
// Used to get only called upon NSM connection close when the network interface
// was still available in the kernel. Due to improvements around interfaceMonitor,
// also expect a callback on interface remove events originating from kernel (only
// interface index and name will be set).
//
// Note: IMHO there's no need for a callback with missing interface index upon
// NSM connection close, since either the interface exists during close, or it does
// not. And in the latter case a kernel event is supposed to take care of unlinking.
// (Therefore, no need for an additional callback that contains only address info.)
// Moreover, this way we don't have to deal with InterfaceDeleted events being
// spammed by NSM heal when reconnect attempt fails and heal orders close of
// non-established connection.
//
// Note: I'm a bit puzzled about abrupt proxy container restart. Although the proxy
// won't remember the next hop addresses. Yet, the associated routes might remain in the
// kernel until the first next hop address is learnt to update the shared routing table.
// This might even be beneficial as such route can provide continuity of egress
// communication assuming the associated LB is up. (Old interfaces remain in POD
// until NSM cleans up the related connections.)
//
// Note: Might get called parallel for multiple NSC connections for example during
// NSM heal, so locking of resources is required.
func (p *Proxy) InterfaceDeleted(intf networking.Iface) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// For kernel originated event lookup linked interface based on the index.
	// And continue processing with the stored Iface.
	fromKernel := false
	if !p.isNSMInterface(intf) && intf.GetIndex() >= 0 {
		linkedIntf := p.Bridge.FindLinkedInterfaceByIndex(intf.GetIndex())
		if linkedIntf == nil {
			return
		}
		p.logger.V(1).Info("InterfaceDeleted event from kernel",
			"name", intf.GetName(networking.WithNoLoad()), "index", intf.GetIndex())
		fromKernel = true
		intf = linkedIntf
	}
	if !p.isNSMInterface(intf) {
		return
	}
	p.logger.Info("InterfaceDeleted", "name", intf.GetName(networking.WithNoLoad()),
		"index", intf.GetIndex(), "intf", intf, "nexthops", p.nexthops)
	// Unlink the interface from the bridge
	err := p.Bridge.UnLinkInterface(intf)
	if err != nil {
		// avoid unnecessary errors in case of kernel reported interface unavailable
		var linkNotFoundErr netlink.LinkNotFoundError
		if !fromKernel || !errors.As(err, &linkNotFoundErr) {
			p.logger.Error(err, "UnLinkInterface")
		}
	}
	if intf.GetInterfaceType() == networking.NSC {
		// 	Remove the neighbor IPs of the interface from the nexthops (outgoing traffic)
		for _, ip := range intf.GetNeighborPrefixes() {
			_, isKnownNexthop := p.nexthops[ip]
			err = p.nexthopRoute.RemoveNexthop(ip)
			if err != nil && isKnownNexthop {
				p.logger.Error(err, "Removing nexthop")
			}

			delete(p.nexthops, ip)
		}
	}
}

// SetIPContext
// XXX: What should we do about new connection establishment requests that fail,
// and thus allocated IP addresses might be leaked if the originating NSC gives up
// for some reason? On the other hand, would there by any unwanted side-effects of
// calling UnsetIPContext() from ipcontext Server/Client when Request to establish
// a new connection fails? NSM retry clients like fullMeshClient clones the request
// on each new try, thus won't cache any assigned IPs. However, NSM heal with reselect
// seems weird, as it keeps Closing and re-requeting the connection including the old
// IPs until it succeeds to reconnect or the "user" closes the connection. Thus, due
// to heal (with reconnect) the IPs might get updated in the NSC case, if someone
// happanned to allocat them between two reconnect attempts. This doesn't seem to be
// a problem, as it should update the connection accordingly. Based on the above,
// IMHO it would make sense releasing allocated IPs by ipcontext Server/Client upon
// unsuccesful Requests where NSM connection was not established. In the server case
// though, it's unlikely that the Request would fail at the proxy.
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

	// cancels the ip release for this connection if it is in progress
	p.connectionToReleaseMutex.Lock()
	cancel, exists := p.connectionToReleaseMap[id]
	if exists {
		p.logger.V(1).Info("Cancel IP release", "id", id)
		cancel()
	}
	p.connectionToReleaseMutex.Unlock()

	srcIPAddrs := []string{}
	dstIpAddrs := []string{}
	// note: If IPAM was not reachable then "user" might not receive the error Allocate
	// returned without a custom context with (suitable) timeout.
	// TODO: could be handy to be able and infer if Allocate() had to reserve an address
	// TODO: NSM retry client like fullMeshClient
	// TODO: If an allocate failed but some have succeeded before, then the allocated
	// IPs might be leaked in case the NSM connection was not established and the user
	// gave up (note: interfaceType == networking.NSC has been covered).
	for _, subnet := range p.Subnets {
		child := &ipamAPI.Child{
			Name:   fmt.Sprintf("%s%s", id, srcChildNamePrefix),
			Subnet: subnet,
		}
		srcPrefix, err := p.IpamClient.Allocate(ctx, child)
		if err != nil {
			return fmt.Errorf("proxy failed to allocate IP for child %v: %w", child, err)
		}
		srcIPAddrs = append(srcIPAddrs, srcPrefix.ToString())

		child = &ipamAPI.Child{
			Name:   fmt.Sprintf("%s%s", id, dstChildNamePrefix),
			Subnet: subnet,
		}
		dstPrefix, err := p.IpamClient.Allocate(ctx, child)
		if err != nil {
			return fmt.Errorf("proxy failed to allocate IP for child %v: %w", child, err)
		}
		dstIpAddrs = append(dstIpAddrs, dstPrefix.ToString())
	}

	if interfaceType == networking.NSE {
		p.setNSEIpContext(id, conn.GetContext().GetIpContext(), srcIPAddrs, dstIpAddrs)
	} else if interfaceType == networking.NSC {
		ipContext := conn.GetContext().GetIpContext()
		oldSrcIpAddrs := ipContext.SrcIpAddrs
		oldDstIpAddrs := ipContext.DstIpAddrs
		ipContext.SrcIpAddrs = dstIpAddrs
		ipContext.DstIpAddrs = srcIPAddrs
		// Note: It might be confusing to see all the "release IP" msgs if the
		// LB NSE is gone, but NSM Find Client haven't reported it yet in order
		// to close the related connection.
		if len(oldSrcIpAddrs) == 0 && len(oldDstIpAddrs) == 0 {
			// src and dst IP addresses were not filled in the request
			// note: NSM retry clients like fullMeshClient clone the Request
			// upon each retry after failed connection establishment, thus
			// src/dst information are empty.
			p.logger.V(1).Info("Set IP Context of initial connection request",
				"id", id, "ipContext", ipContext, "interfaceType", "NSC")
		} else {
			if !utils.EqualStringList(oldSrcIpAddrs, dstIpAddrs) || !utils.EqualStringList(oldDstIpAddrs, srcIPAddrs) {
				// IMHO update of IP addresses should be avoided for interfaces
				// of type NSC. That's because if the interface was not removed
				// prior to this to trigger an InterfaceDeleted event, then the
				// proxy would not remove the remote IPs from the list of valid
				// nexthops (possibly resulting in invalid routes).
				p.logger.Info("Updated IP Context of connection request",
					"id", id, "ipContext", ipContext, "interfaceType", "NSC",
					"oldSrcIPs", oldSrcIpAddrs, "oldDstIPs", oldDstIpAddrs)
			}
		}
	}
	return nil
}

func (p *Proxy) setNSEIpContext(id string, ipContext *networkservice.IPContext, srcIPAddrs []string, dstIpAddrs []string) {
	if len(ipContext.SrcIpAddrs) == 0 && len(ipContext.DstIpAddrs) == 0 { // First request
		ipContext.SrcIpAddrs = srcIPAddrs
		ipContext.DstIpAddrs = dstIpAddrs
		ipContext.ExtraPrefixes = p.Bridge.GetLocalPrefixes()
		p.logger.V(1).Info("Set IP Context of initial connection request",
			"id", id, "ipContext", ipContext, "interfaceType", "NSE")
		return
	}
	// The request is most probably an update
	// But it might be a continuation of a pre-existing connection
	// from a TAPA established with an old proxy instance. Could be worth
	// verifying gateways (might change in theory e.g. when trench/conduit
	// was removed and re-deployed).
	if contains(ipContext.ExtraPrefixes, p.Bridge.GetLocalPrefixes()) &&
		contains(ipContext.SrcIpAddrs, srcIPAddrs) &&
		contains(ipContext.DstIpAddrs, dstIpAddrs) {
		if !contains(ipContext.GetExtraPrefixes(), p.Bridge.GetLocalPrefixes()) {
			// set the gateways
			oldGateways := ipContext.GetExtraPrefixes()
			ipContext.ExtraPrefixes = p.Bridge.GetLocalPrefixes()
			p.updatePolicyRoutes(ipContext) // update policy routes
			p.logger.Info("Updated IP Context of connection request", "id", id,
				"ipContext", ipContext, "oldGateways", oldGateways,
				"interfaceType", "NSE",
			)
		}
		return
	}
	// remove old IPs, add new ones, and set the gateways
	oldSrcIpAddrs := ipContext.SrcIpAddrs
	oldDstIpAddrs := ipContext.DstIpAddrs
	oldGateways := ipContext.GetExtraPrefixes()
	ipContext.ExtraPrefixes = p.Bridge.GetLocalPrefixes()
	ipContext.SrcIpAddrs = removeOldIPs(ipContext.SrcIpAddrs, oldGateways)
	ipContext.DstIpAddrs = removeOldIPs(ipContext.DstIpAddrs, oldGateways)
	ipContext.SrcIpAddrs = append(ipContext.SrcIpAddrs, srcIPAddrs...)
	ipContext.DstIpAddrs = append(ipContext.DstIpAddrs, dstIpAddrs...)
	p.updatePolicyRoutes(ipContext) // update policy routes
	p.logger.Info("Updated IP Context of connection request", "id", id,
		"ipContext", ipContext, "oldSrcIPs", oldSrcIpAddrs,
		"oldDstIPs", oldDstIpAddrs, "oldGateways", oldGateways,
		"interfaceType", "NSE",
	)
}

func (p *Proxy) updatePolicyRoutes(ipContext *networkservice.IPContext) {
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

func (p *Proxy) UnsetIPContext(ctx context.Context, conn *networkservice.Connection,
	interfaceType networking.InterfaceType, delay time.Duration) error {
	id := conn.Id
	if interfaceType == networking.NSE {
		// Use the segment ID referring to the TAPA side of the NSM connection
		id = conn.GetPath().GetPathSegments()[0].Id
	}
	// Release the IPs in background, so it is not blocking in case the IPAM is down
	go p.ipReleaser(id, delay)
	return nil
}

func (p *Proxy) ipReleaser(id string, delay time.Duration) {
	p.connectionToReleaseMutex.Lock()
	_, exists := p.connectionToReleaseMap[id]
	if exists { // If an ipReleaser is already running for this connection Id
		p.connectionToReleaseMutex.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.connectionToReleaseMap[id] = cancel // So SetIPContext can cancel in case it needs the IP for this connection
	p.logger.V(1).Info("Attempt IP release", "id", id, "delay", delay)
	p.connectionToReleaseMutex.Unlock()

	if delay > 0 { // Delay release
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			select {
			case <-t.C:
			default:
			}
			p.logger.V(1).Info("Canceled delayed IP release", "id", id)
		case <-t.C: // delay before attempting actual release
		}
	}

	select {
	case <-ctx.Done():
	default:
		_ = retry.Do(func() error {
			ctxRelease, cancelRelease := context.WithTimeout(ctx, 10*time.Second)
			defer cancelRelease()
			for _, subnet := range p.Subnets {
				child := &ipamAPI.Child{
					Name:   fmt.Sprintf("%s%s", id, srcChildNamePrefix),
					Subnet: subnet,
				}
				_, err := p.IpamClient.Release(ctxRelease, child)
				if err != nil {
					if ctxRelease.Err() != context.Canceled {
						p.logger.Error(err, "failed to release src IP", "id", id, "subnet", subnet)
					}
					return fmt.Errorf("proxy failed to release IP for child %v, %w", child, err)
				}
				child = &ipamAPI.Child{
					Name:   fmt.Sprintf("%s%s", id, dstChildNamePrefix),
					Subnet: subnet,
				}
				_, err = p.IpamClient.Release(ctxRelease, child)
				if err != nil {
					if ctxRelease.Err() != context.Canceled {
						p.logger.Error(err, "failed to release dst IP", "id", id, "subnet", subnet)
					}
					return fmt.Errorf("proxy failed to release IP for child %v, %w", child, err)
				}
				p.logger.Info("release IP", "id", id, "subnet", subnet)
			}
			return nil
		}, retry.WithContext(ctx),
			retry.WithDelay(2*time.Second))
	}
	p.connectionToReleaseMutex.Lock()
	delete(p.connectionToReleaseMap, id)
	p.connectionToReleaseMutex.Unlock()
}

func (p *Proxy) setBridgeIP(prefix string) error {
	err := p.Bridge.AddLocalPrefix(prefix)
	if err != nil {
		return fmt.Errorf("failed to set bridge IP %s, %w", prefix, err)
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
			return fmt.Errorf("failed to allocate bridge IP for child %v: %w", child, err)
		}
		err = p.setBridgeIP(prefix.ToString())
		if err != nil {
			return err
		}
		p.logger.Info("Set bridge IP", "IP", prefix.ToString(), "child", child)
	}
	return nil
}

// cleanupRules flushes any potentially lingering IP rules that might remain
// after a container or process restart.
// Note: In theory newVirtualIP() could double-check if there was a rule already
// installed for the particular VIP address but referencing a different table.
// But due to the dynamic nature of vips (might change during proxy restart),
// it's safer to flush rules on start.
func (p *Proxy) cleanupRules(ctx context.Context) {
	logger := p.logger.WithValues("func", "cleanupRules")
	getNetlinkFamily := func(subnet *ipamAPI.Subnet) int {
		switch subnet.IpFamily {
		case ipamAPI.IPFamily_IPV4:
			return netlink.FAMILY_V4
		case ipamAPI.IPFamily_IPV6:
			return netlink.FAMILY_V6
		default:
			return netlink.FAMILY_ALL
		}
	}

	for _, subnet := range p.Subnets {
		err := flushRules(ctx, getNetlinkFamily(subnet), tableID)
		if err != nil {
			logger.Error(err, "Failed to flush rules for subnet", "subnet", subnet)
		} else {
			logger.Info("Successfully flushed rules for subnet", "subnet", subnet)
		}
	}
}

func (p *Proxy) SetVIPs(vips []string) {
	once.Do(func() {
		// Cleaning up source based routing rules that might have persisted in the
		// network stack after a container/process restart. Performing the cleanup
		// here aims to minimize traffic disturbance once NSM datapath monitoring
		// is enabled (allowing NSM intefaces to survive such events). Currently,
		// the rule cleanup could be safely executed during NewProxy() call.
		p.cleanupRules(context.Background())
	})

	p.mu.Lock()
	defer p.mu.Unlock()

	logger := p.logger.WithValues("func", "SetVIPs")
	currentVIPs := make(map[string]*virtualIP)
	for _, vip := range p.vips {
		currentVIPs[vip.prefix] = vip
	}
	for _, vip := range vips {
		if _, ok := currentVIPs[vip]; !ok {
			logger.Info("Add VIP", "vip", vip)
			newVIP, err := newVirtualIP(vip, tableID, p.netUtils) // use shared tableID to lookup nexthop route for outbound VIP traffic
			if err != nil {
				logger.Error(err, "Adding SourceBaseRoute", "vip", vip, "tableID", tableID)
				continue
			}
			p.vips = append(p.vips, newVIP)
			// note: don't add nexthops to vips; the "nexthop" routing table is managed independently
		}
		delete(currentVIPs, vip)
	}
	// delete remaining vips
	for index := 0; index < len(p.vips); index++ {
		vip := p.vips[index]
		if _, ok := currentVIPs[vip.prefix]; ok {
			logger.Info("Delete VIP", "vip", vip.prefix)
			p.vips = append(p.vips[:index], p.vips[index+1:]...)
			index--
			err := vip.Delete()
			if err != nil {
				logger.Error(err, "Deleting vip", "vip", vip.prefix)
			}
		}
	}
}

// Close -
// Tries to force release of IPs with pending release
func (p *Proxy) Close(ctx context.Context) {
	logger := p.logger.WithValues("func", "Close")
	var wg sync.WaitGroup

	p.connectionToReleaseMutex.Lock()
	logger.Info("Release pending IPs", "num", len(p.connectionToReleaseMap))
	defer logger.Info("Pending IP release concluded")

	wg.Add(len(p.connectionToReleaseMap))
	for id, cancel := range p.connectionToReleaseMap {
		cancel()
		go func(ctx context.Context, id string) {
			defer wg.Done()

			_ = retry.Do(func() error {
				ctxRelease, cancelRelease := context.WithTimeout(ctx, 4*time.Second)
				defer cancelRelease()
				for _, subnet := range p.Subnets {
					child := &ipamAPI.Child{
						Name:   fmt.Sprintf("%s%s", id, srcChildNamePrefix),
						Subnet: subnet,
					}
					_, err := p.IpamClient.Release(ctxRelease, child)
					if err != nil {
						logger.Error(err, "failed to release src IP", "id", id, "subnet", subnet)
						return fmt.Errorf("proxy failed to release IP for child %v, %w", child, err)
					}
					child = &ipamAPI.Child{
						Name:   fmt.Sprintf("%s%s", id, dstChildNamePrefix),
						Subnet: subnet,
					}
					_, err = p.IpamClient.Release(ctxRelease, child)
					if err != nil {
						logger.Error(err, "failed to release dst IP", "id", id, "subnet", subnet)
						return fmt.Errorf("proxy failed to release IP for child %v, %w", child, err)
					}
					logger.Info("release IP", "id", id, "subnet", subnet)
				}
				return nil
			}, retry.WithContext(ctx),
				retry.WithDelay(200*time.Millisecond))
		}(ctx, id)
	}

	p.connectionToReleaseMutex.Unlock()
	wg.Wait()
}

// NewProxy -
func NewProxy(conduit *nspAPI.Conduit, nodeName string, ipamClient ipamAPI.IpamClient, ipFamily string, netUtils networking.Utils) *Proxy {
	logger := log.Logger.WithValues("class", "Proxy")
	bridge, err := netUtils.NewBridge("bridge0")
	if err != nil {
		logger.Error(err, "Creating the bridge")
	}
	proxy := &Proxy{
		Bridge:                 bridge,
		conduit:                conduit,
		netUtils:               netUtils,
		nexthops:               map[string]struct{}{},
		vips:                   []*virtualIP{},
		nexthopRoute:           kernel.NewNexthopRoute(tableID),
		Subnets:                make(map[ipamAPI.IPFamily]*ipamAPI.Subnet),
		IpamClient:             ipamClient,
		logger:                 logger,
		connectionToReleaseMap: map[string]context.CancelFunc{},
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
	// TODO: Consider removing bridge interface or changing its state to DOWN on teardown:
	// During upgrade tests when running with vpp-forwarders where hostPID=false, some
	// TAPAs seemed to remain connected with some "old" proxy instance (according to the
	// ARP entries on the TAPA side; ping worked!). If opting for changing the interface
	// state DOWN, the bridge create function should be modified to make sure the state
	// is UP in case the bridge exists.)
	err = proxy.setBridgeIPs()
	if err != nil {
		logger.Error(err, "Setting the bridge IP")
	}
	return proxy
}

// FlushRules lists and deletes IP rules for the specified family and routing table.
// Errors encountered during rule deletion are accumulated and returned.
func flushRules(ctx context.Context, family int, table int) error {

	ruleFilter := &netlink.Rule{Table: table}
	filterMask := netlink.RT_FILTER_TABLE

	rules, err := netlink.RuleListFiltered(family, ruleFilter, filterMask)
	if err != nil {
		return fmt.Errorf("failed to fetch ip rules: %w", err)
	}

	logger := log.FromContextOrGlobal(ctx).WithName("FlushRules")
	for _, rule := range rules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:

		}
		if rule.Family == 0 {
			rule.Family = family // explicitly set family, so that IPv6 "from any" rules can be also removed
		}
		logger.V(1).Info("Delete IP rule", "family", rule.Family, "rule", rule)
		delErr := netlink.RuleDel(&rule)
		if delErr != nil {
			err = utils.AppendErr(err, fmt.Errorf("failed to delete ip rule %v, %w", rule, delErr))
		}
	}
	return err
}
