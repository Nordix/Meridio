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

package interfacemonitor

import (
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
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
	index, err := im.netUtils.GetIndexFromName(conn.getInterfaceName())
	if err != nil {
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
func (im *interfaceMonitor) InterfaceDeleted(intf networking.Iface) {
}

func (im *interfaceMonitor) advertiseInterfaceCreation(index int, pendingInterface *pendingInterface) {
	newInterface := im.netUtils.NewInterface(index)
	newInterface.SetLocalPrefixes(pendingInterface.localIPs)
	newInterface.SetNeighborPrefixes(pendingInterface.neighborIPs)
	newInterface.SetGatewayPrefixes(pendingInterface.gateways)
	newInterface.SetInterfaceType(pendingInterface.InterfaceType)
	logrus.Debugf("interfaceMonitor: advertise created intf %v", newInterface)
	im.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

func (im *interfaceMonitor) advertiseInterfaceDeletion(intf networking.Iface) {
	logrus.Debugf("interfaceMonitor: advertise deleted intf %v", intf)
	im.interfaceMonitorSubscriber.InterfaceDeleted(intf)
}

func newInterfaceMonitor(networkInterfaceMonitor networking.InterfaceMonitor, interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networkingUtils) *interfaceMonitor {
	im := &interfaceMonitor{
		networkInterfaceMonitor:    networkInterfaceMonitor,
		interfaceMonitorSubscriber: interfaceMonitorSubscriber,
		netUtils:                   netUtils,
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
