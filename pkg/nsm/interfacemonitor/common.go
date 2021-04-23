package interfacemonitor

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/nordix/meridio/pkg/networking"
)

type ipList []string

type connection struct {
	*networkservice.Connection
}

type interfaceMonitor struct {
	interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber
	netUtils                   networking.Utils
	pendingInterfaces          map[string]*pendingInterface
}

type pendingInterface struct {
	interfaceName string
	localIPs      ipList
	neighborIPs   ipList
	gateways      ipList
	InteraceType  networking.InteraceType
}

func (im *interfaceMonitor) ConnectionRequested(conn *connection, interaceType networking.InteraceType) {
	interfaceName := conn.getInterfaceName()
	pendingInterface := &pendingInterface{
		interfaceName: interfaceName,
		localIPs:      conn.getLocalIPs(),
		neighborIPs:   conn.getNeighborIPs(),
		gateways:      conn.getGatewayIPs(),
	}

	index, err := im.netUtils.GetIndexFromName(interfaceName)
	if err == nil {
		im.advertiseInterfaceCreation(index, pendingInterface)
	} else {
		im.pendingInterfaces[interfaceName] = pendingInterface
	}
}

func (im *interfaceMonitor) ConnectionClosed(conn *connection, interaceType networking.InteraceType) {
	index, err := im.netUtils.GetIndexFromName(conn.getInterfaceName())
	if err != nil {
		return
	}
	im.advertiseInterfaceDeletion(index, interaceType)
}

// InterfaceCreated -
func (im *interfaceMonitor) InterfaceCreated(intf networking.Iface) {
	if nsmInterface, ok := im.pendingInterfaces[intf.GetName()]; ok {
		delete(im.pendingInterfaces, intf.GetName())
		im.advertiseInterfaceCreation(intf.GetIndex(), nsmInterface)
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
	newInterface.SetInteraceType(pendingInterface.InteraceType)
	im.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

func (im *interfaceMonitor) advertiseInterfaceDeletion(index int, interaceType networking.InteraceType) {
	newInterface := im.netUtils.NewInterface(index)
	newInterface.SetInteraceType(interaceType)
	im.interfaceMonitorSubscriber.InterfaceDeleted(newInterface)
}

func NewInterfaceMonitor(interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networking.Utils) *interfaceMonitor {
	return &interfaceMonitor{
		interfaceMonitorSubscriber: interfaceMonitorSubscriber,
		pendingInterfaces:          make(map[string]*pendingInterface),
		netUtils:                   netUtils,
	}
}

func (conn *connection) getInterfaceName() string {
	if conn == nil || conn.GetMechanism() == nil || conn.GetMechanism().GetParameters() == nil {
		return ""
	}
	return conn.GetMechanism().GetParameters()[common.InterfaceNameKey]
}

func (conn *connection) getLocalIPs() []string {
	localIPs := []string{}
	if conn == nil || conn.GetContext() == nil || conn.GetContext().GetIpContext() == nil {
		return localIPs
	}
	localIPs = append(localIPs, conn.GetContext().GetIpContext().GetDstIpAddr())
	return localIPs
}

func (conn *connection) getNeighborIPs() []string {
	neighborIPs := []string{}
	if conn == nil || conn.GetContext() == nil || conn.GetContext().GetIpContext() == nil {
		return neighborIPs
	}
	neighborIPs = append(neighborIPs, conn.GetContext().GetIpContext().GetSrcIpAddr())
	return neighborIPs
}

func (conn *connection) getGatewayIPs() []string {
	if conn == nil || conn.GetContext() == nil || conn.GetContext().GetIpContext() == nil {
		return []string{}
	}
	return conn.GetContext().GetIpContext().ExtraPrefixes
}
