/*
Copyright (c) 2021-2023 Nordix Foundation

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

package interfacemonitor

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
)

type ipList []string

type networkingUtils interface {
	NewInterface(index int) networking.Iface
	GetIndexFromName(name string) (int, error)
}

type connection struct {
	*networkservice.Connection
}

type interfaceMonitor struct {
	networkInterfaceMonitor    networking.InterfaceMonitor
	interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber
	netUtils                   networkingUtils
	pendingInterfaces          sync.Map // map[string]*pendingInterface
	logger                     logr.Logger
}

type pendingInterface struct {
	interfaceName string
	localIPs      ipList
	neighborIPs   ipList
	gateways      ipList
	InterfaceType networking.InterfaceType
}

// ConnectionRequested -
// Note: The pendingInterfaces logic relies on the interface name to relate kernel
// and NSM originated events to collect all the info necessary for notifying subscribers.
func (im *interfaceMonitor) ConnectionRequested(conn *connection, interfaceType networking.InterfaceType) {
	interfaceName := conn.getInterfaceName()
	pendingInterface := &pendingInterface{
		interfaceName: interfaceName,
		gateways:      conn.getGatewayIPs(),
		InterfaceType: interfaceType,
	}
	if interfaceType == networking.NSC {
		pendingInterface.localIPs = conn.getSrcIPs()
		pendingInterface.neighborIPs = conn.getDstIPs()
	} else if interfaceType == networking.NSE {
		pendingInterface.localIPs = conn.getDstIPs()
		pendingInterface.neighborIPs = conn.getSrcIPs()
	}

	index, err := im.netUtils.GetIndexFromName(interfaceName)
	if err == nil {
		im.advertiseInterfaceCreation(index, pendingInterface)
	} else {
		im.pendingInterfaces.Store(interfaceName, pendingInterface)
	}
}

func (im *interfaceMonitor) ConnectionClosed(conn *connection, interfaceType networking.InterfaceType) {
	name := conn.getInterfaceName()
	if name == "" { // non-established connection have no interface name
		return
	}
	index, err := im.netUtils.GetIndexFromName(name)
	if err != nil {
		// XXX: In case of interfaceType NSE, the interface is normally no
		// longer available. Might be worth giving it a try with an invalid
		// index, relying on the name of the inteface and other available info.
		// Although the interface name is not passed on creation. Why not?
		if interfaceType == networking.NSC || interfaceType == networking.NSE {
			id := conn.GetId()
			if interfaceType == networking.NSE && conn.GetPath() != nil &&
				len(conn.GetPath().GetPathSegments()) >= 1 {
				id = conn.GetPath().GetPathSegments()[0].GetId()
			}
			// log the ID of "nsc" path segment for visibility
			im.logger.V(1).Info("no interface index on connection close",
				"id", id, "interfaceType", interfaceType, "err", err)
		}
		// XXX: Instead of return, maybe consider informing users even without a valid
		// interface index. (IMHO not needed assuming kernel originated delete event
		// is handled correctly. Don't see any added value in case of process/container
		// crash either.)
		// Note: NSM heal might spam ConnectionClosed events when it tries to reconnect
		// with no luck. Thus, without returning, NSM heal would also be a source of
		// false positive interface delete advertisements.
		return
	}

	newInterface := im.netUtils.NewInterface(index)
	newInterface.SetInterfaceType(interfaceType)
	newInterface.SetGatewayPrefixes(conn.getGatewayIPs())
	if interfaceType == networking.NSC {
		newInterface.SetLocalPrefixes(conn.getSrcIPs())
		newInterface.SetNeighborPrefixes(conn.getDstIPs())
	} else if interfaceType == networking.NSE {
		newInterface.SetLocalPrefixes(conn.getDstIPs())
		newInterface.SetNeighborPrefixes(conn.getSrcIPs())
	}

	im.advertiseInterfaceDeletion(newInterface)
}

// InterfaceCreated -
func (im *interfaceMonitor) InterfaceCreated(intf networking.Iface) {
	if nsmInterface, ok := im.pendingInterfaces.Load(intf.GetName()); ok {
		im.pendingInterfaces.Delete(intf.GetName())
		im.advertiseInterfaceCreation(intf.GetIndex(), nsmInterface.(*pendingInterface))
	}
}

// InterfaceDeleted -
// InterfaceDeleted forwards kernel originated interface delete events.
//
// Unlike during create, here there's no "cache" to lookup all the information
// required for creating a fully fledged networking.Interface item.
// Now, we have both the name and index in case of kernel originated delete event.
// And it seems unlikely to match index (and/or name) while the subsriber was aware
// of an existing interface with the same parameter, yet would pertain to a "different"
// interface. So, it looks safe to fire an event solely based on inteface index and
// name. And it should be up to the subscriber to either ignore or process it (e.g.
// do a lookup based on interface index and/or name).
//
// Note: Only Proxy and LB uses interfaceMonitor. (LB's usage of InterfaceDeleted
// events is rather simple and has been using kernel originated events for some time.)
// The Proxy has both NSC and NSE roles. In case of NSE role, the kernel takes care of
// detaching the removed interface from the linux bridge by default. Seemingly no other
// reconfiguration is required in that case. Meaning, even if interface removal was not
// advertised (not even by connection close), things wouldn't break.
// When Proxy acts as a NSC (i.e. connects to LBs) it must get informed about the loss
// of the network interface had it been removed before the connection close. (So that it
// could reconfigure its nexthop routes.)
//
// Note: The subscriber should be prepared to handle interface delete events with missing
// information, when only the interface index (and interface name) are known.
// Note: interfacemonitor.server and client will both create an InterfaceMonitor, thus
// such events will be received twice by the Proxy
func (im *interfaceMonitor) InterfaceDeleted(intf networking.Iface) {
	im.advertiseInterfaceDeletion(intf) // interface event solely based on interface index/name
}

func (im *interfaceMonitor) advertiseInterfaceCreation(index int, pendingInterface *pendingInterface) {
	newInterface := im.netUtils.NewInterface(index)
	newInterface.SetLocalPrefixes(pendingInterface.localIPs)
	newInterface.SetNeighborPrefixes(pendingInterface.neighborIPs)
	newInterface.SetGatewayPrefixes(pendingInterface.gateways)
	newInterface.SetInterfaceType(pendingInterface.InterfaceType)
	im.logger.V(1).Info("advertise created", "interface", newInterface, "index", newInterface.GetIndex())
	im.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

func (im *interfaceMonitor) advertiseInterfaceDeletion(intf networking.Iface) {
	im.logger.V(1).Info("advertise deleted", "interface", intf, "index", intf.GetIndex())
	im.interfaceMonitorSubscriber.InterfaceDeleted(intf)
}

func newInterfaceMonitor(networkInterfaceMonitor networking.InterfaceMonitor, interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networkingUtils) *interfaceMonitor {
	im := &interfaceMonitor{
		networkInterfaceMonitor:    networkInterfaceMonitor,
		interfaceMonitorSubscriber: interfaceMonitorSubscriber,
		netUtils:                   netUtils,
		logger:                     log.Logger.WithValues("class", "interfaceMonitor"),
	}
	if networkInterfaceMonitor != nil {
		networkInterfaceMonitor.Subscribe(im)
	}
	return im
}

func (conn *connection) getInterfaceName() string {
	if conn == nil || conn.GetMechanism() == nil || conn.GetMechanism().GetParameters() == nil {
		return ""
	}
	return conn.GetMechanism().GetParameters()[common.InterfaceNameKey]
}

func (conn *connection) getDstIPs() []string {
	if conn == nil || conn.GetContext() == nil || conn.GetContext().GetIpContext() == nil {
		return []string{}
	}
	return conn.GetContext().GetIpContext().GetDstIpAddrs()
}

func (conn *connection) getSrcIPs() []string {
	if conn == nil || conn.GetContext() == nil || conn.GetContext().GetIpContext() == nil {
		return []string{}
	}
	return conn.GetContext().GetIpContext().GetSrcIpAddrs()
}

func (conn *connection) getGatewayIPs() []string {
	if conn == nil || conn.GetContext() == nil || conn.GetContext().GetIpContext() == nil {
		return []string{}
	}
	return conn.GetContext().GetIpContext().ExtraPrefixes
}
