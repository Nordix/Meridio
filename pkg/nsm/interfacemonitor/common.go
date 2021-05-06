package interfacemonitor

import (
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
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
	im.advertiseInterfaceDeletion(index, interfaceType)
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
	im.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

func (im *interfaceMonitor) advertiseInterfaceDeletion(index int, interfaceType networking.InterfaceType) {
	newInterface := im.netUtils.NewInterface(index)
	newInterface.SetInterfaceType(interfaceType)
	im.interfaceMonitorSubscriber.InterfaceDeleted(newInterface)
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
