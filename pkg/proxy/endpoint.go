package proxy

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type ProxyEndpoint struct {
	temporaryNSMInterfaces     map[string]*TemporaryNSMInterface
	interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber
	nseConnectionFactory       endpoint.NSEConnectionFactory
}

type TemporaryNSMInterface struct {
	interfaceName string
	localIPs      []*netlink.Addr
	neighborIPs   []*netlink.Addr
}

func NewProxyEndpoint(interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, nseConnectionFactory endpoint.NSEConnectionFactory) *ProxyEndpoint {
	return &ProxyEndpoint{
		interfaceMonitorSubscriber: interfaceMonitorSubscriber,
		temporaryNSMInterfaces:     make(map[string]*TemporaryNSMInterface),
		nseConnectionFactory:       nseConnectionFactory,
	}
}

func (pe *ProxyEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	ipContext, err := pe.nseConnectionFactory.NewNSEIPContext()
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err creating new IP context: %v", err)
	}
	request.GetConnection().GetContext().IpContext = ipContext

	localIP, err := netlink.ParseAddr(request.GetConnection().GetContext().GetIpContext().DstIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing local IP: %v", err)
	}
	// neighborIP, err := netlink.ParseAddr(request.GetConnection().GetContext().GetIpContext().SrcIpAddr)
	// if err != nil {
	// 	logrus.Errorf("ProxyEndpoint: err parsing neighbor IP: %v", err)
	// }

	// TODO name generation
	randomID := rand.Intn(1000)
	interfaceName := "nse" + strconv.Itoa(randomID)
	logrus.Infof("ProxyEndpoint: interface name: %v", interfaceName)
	request.GetConnection().Mechanism.GetParameters()[kernel.InterfaceNameKey] = interfaceName

	localIPs := []*netlink.Addr{localIP}
	// neighborIPs := []*netlink.Addr{neighborIP}
	neighborIPs := []*netlink.Addr{}

	temporaryNSMInterface := &TemporaryNSMInterface{
		interfaceName: interfaceName,
		localIPs:      localIPs,
		neighborIPs:   neighborIPs,
	}

	index, err := networking.GetIndexFromName(interfaceName)
	if err == nil {
		pe.advertiseInterfaceCreation(index, temporaryNSMInterface)
	} else {
		pe.temporaryNSMInterfaces[interfaceName] = temporaryNSMInterface
	}

	log.FromContext(ctx).Infof("ProxyEndpoint: (Request) temporaryNSMInterface: %+v", temporaryNSMInterface)

	return next.Server(ctx).Request(ctx, request)
}

func (pe *ProxyEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {

	localIP, err := netlink.ParseAddr(conn.GetContext().GetIpContext().DstIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing local IP: %v", err)
	}
	neighborIP, err := netlink.ParseAddr(conn.GetContext().GetIpContext().SrcIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing neighbor IP: %v", err)
	}

	temporaryNSMInterface := &TemporaryNSMInterface{
		interfaceName: conn.Mechanism.Parameters["name"],
		localIPs:      []*netlink.Addr{localIP},
		neighborIPs:   []*netlink.Addr{neighborIP},
	}
	log.FromContext(ctx).Infof("ProxyEndpoint: (Close) temporaryNSMInterface: %+v", temporaryNSMInterface)

	return next.Server(ctx).Close(ctx, conn)
}

func (pe *ProxyEndpoint) advertiseInterfaceCreation(index int, nsmInterface *TemporaryNSMInterface) {
	// TODO: Waiting 2 second until the network interface is created
	go func() {
		time.Sleep(2 * time.Second)
		newInterface := networking.NewInterface(index, nsmInterface.localIPs, nsmInterface.neighborIPs)
		newInterface.InteraceType = networking.NSE
		pe.interfaceMonitorSubscriber.InterfaceCreated(newInterface)
	}()
}

func (pe *ProxyEndpoint) InterfaceCreated(intf *networking.Interface) {
	if nsmInterface, ok := pe.temporaryNSMInterfaces[intf.GetName()]; ok {
		delete(pe.temporaryNSMInterfaces, intf.GetName())
		pe.advertiseInterfaceCreation(intf.GetIndex(), nsmInterface)
	}
}

func (pe *ProxyEndpoint) InterfaceDeleted(intf *networking.Interface) {
}
