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
//
// XXX: There's no "cache" to lookup all the information required for creating
// a networking.Interface items. But, pendingInterfaces relies on the interface
// name as well to link kernel and NSM originated create events to gather all
// the necessary info. Now, we have both the name and index in case of kernel
// originated delete event. IMHO it's unlikely to match both index and name
// while the subsriber was aware of an existing interface with the same two
// parameters yet pertained to a different interface. Also, implementing a
// "pending delete" feature would most probably rely on the exact same concept
// pendingInterfaces uses, that is to match information from two sources based
// on the interface name. So, question comes, why not simply use these two
// parameters to fire an event and leave it to the subscriber to either ignore
// it, or do a lookup solely based on interface index and name. Luckily, only
// the proxy uses this. Where in case of NSE role, the interface removal will
// automatically detach the interface from the bridge, and doesn't seem to be
// anything else to reconfigure. So, even though interface deletion won't
// be advertised for NSE (not even by connection close), things shall work.
// Also, proxy relies on DeepEqual to compare interfaces received in events
// against stored ones, thus without major changes in the proxy, there wouldn't
// be any point firing an event with missing interface index information...
func (im *interfaceMonitor) InterfaceDeleted(intf networking.Iface) {
}

func (im *interfaceMonitor) advertiseInterfaceCreation(index int, pendingInterface *pendingInterface) {
	newInterface := im.netUtils.NewInterface(index)
	newInterface.SetLocalPrefixes(pendingInterface.localIPs)
	newInterface.SetNeighborPrefixes(pendingInterface.neighborIPs)
	newInterface.SetGatewayPrefixes(pendingInterface.gateways)
	newInterface.SetInterfaceType(pendingInterface.InterfaceType)
	im.logger.V(1).Info("advertise created", "interface", newInterface)
	im.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

func (im *interfaceMonitor) advertiseInterfaceDeletion(intf networking.Iface) {
	im.logger.V(1).Info("advertise deleted", "interface", intf)
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
