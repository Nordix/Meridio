package nsm

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// InterfaceMonitorEndpoint -
type InterfaceMonitorEndpoint struct {
	temporaryNSMInterfaces     map[string]*TemporaryNSMInterface
	interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber
}

// TemporaryNSMInterface -
type TemporaryNSMInterface struct {
	interfaceName string
	localIPs      []*netlink.Addr
	neighborIPs   []*netlink.Addr
}

// NewInterfaceMonitorEndpoint -
func NewInterfaceMonitorEndpoint(interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber) *InterfaceMonitorEndpoint {
	return &InterfaceMonitorEndpoint{
		interfaceMonitorSubscriber: interfaceMonitorSubscriber,
		temporaryNSMInterfaces:     make(map[string]*TemporaryNSMInterface),
	}
}

// Request -
func (ime *InterfaceMonitorEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	localIP, err := netlink.ParseAddr(request.GetConnection().GetContext().GetIpContext().DstIpAddr)
	if err != nil {
		logrus.Errorf("InterfaceMonitorEndpoint: err parsing local IP: %v", err)
	}
	neighborIP, err := netlink.ParseAddr(request.GetConnection().GetContext().GetIpContext().SrcIpAddr)
	if err != nil {
		logrus.Errorf("InterfaceMonitorEndpoint: err parsing neighbor IP: %v", err)
	}

	localIPs := []*netlink.Addr{localIP}
	neighborIPs := []*netlink.Addr{neighborIP}

	interfaceName := request.GetConnection().Mechanism.GetParameters()[kernel.InterfaceNameKey]

	temporaryNSMInterface := &TemporaryNSMInterface{
		interfaceName: interfaceName,
		localIPs:      localIPs,
		neighborIPs:   neighborIPs,
	}

	index, err := networking.GetIndexFromName(interfaceName)
	if err == nil {
		ime.advertiseInterfaceCreation(index, temporaryNSMInterface)
	} else {
		ime.temporaryNSMInterfaces[interfaceName] = temporaryNSMInterface
	}

	return next.Server(ctx).Request(ctx, request)
}

// Close -
func (ime *InterfaceMonitorEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	interfaceName := conn.Mechanism.GetParameters()[kernel.InterfaceNameKey]
	index, err := networking.GetIndexFromName(interfaceName)
	if err == nil {
		ime.advertiseInterfaceDeletion(index)
	}
	return next.Server(ctx).Close(ctx, conn)
}

func (ime *InterfaceMonitorEndpoint) advertiseInterfaceCreation(index int, nsmInterface *TemporaryNSMInterface) {
	newInterface := networking.NewInterface(index, nsmInterface.localIPs, nsmInterface.neighborIPs)
	newInterface.InteraceType = networking.NSE
	ime.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

func (ime *InterfaceMonitorEndpoint) advertiseInterfaceDeletion(index int) {
	newInterface := networking.NewInterface(index, []*netlink.Addr{}, []*netlink.Addr{})
	newInterface.InteraceType = networking.NSE
	ime.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
}

// InterfaceCreated -
func (ime *InterfaceMonitorEndpoint) InterfaceCreated(intf *networking.Interface) {
	if nsmInterface, ok := ime.temporaryNSMInterfaces[intf.GetName()]; ok {
		delete(ime.temporaryNSMInterfaces, intf.GetName())
		ime.advertiseInterfaceCreation(intf.GetIndex(), nsmInterface)
	}
}

// InterfaceDeleted -
func (ime *InterfaceMonitorEndpoint) InterfaceDeleted(intf *networking.Interface) {
}
