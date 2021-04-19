package client

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"

	"github.com/sirupsen/logrus"
)

type NetworkServiceClient struct {
	Id                         string
	NetworkServiceName         string
	NetworkServiceEndpointName string
	Labels                     map[string]string
	ExtraContext               map[string]string
	Connection                 *networkservice.Connection
	nsmgrClient                NSMgrClient
	InterfaceName              string
	InterfaceMonitorSubscriber networking.InterfaceMonitorSubscriber
	intf                       *networking.Interface
	nscConnectionFactory       NSCConnectionFactory
}

type NSMgrClient interface {
	Request(*networkservice.NetworkServiceRequest) (*networkservice.Connection, error)
	Close(*networkservice.Connection) (*empty.Empty, error)
}

// Request
func (nsc *NetworkServiceClient) Request() {
	request := nsc.prepareRequest()
	for {
		var err error
		nsc.Connection, err = nsc.nsmgrClient.Request(request)
		if err != nil {
			time.Sleep(15 * time.Second)
			logrus.Errorf("Network Service Client: Request err: %v", err)
			continue
		}
		nsc.setIntf()
		nsc.advertiseInterfaceCreation()
		break
	}
}

// Close -
func (nsc *NetworkServiceClient) Close() {
	nsc.advertiseInterfaceDeletion()
	// var err error
	// _, err = nsc.nsmgrClient.Close(nsc.Connection)
	// if err != nil {
	// 	logrus.Errorf("Network Service Client: Close err: %v", err)
	// }
}

func (nsc *NetworkServiceClient) setIntf() {
	index, err := networking.GetIndexFromName(nsc.InterfaceName)
	if err != nil {
		logrus.Errorf("Network Service Client: GetIndexFromName err: %v", err)
	}

	localIPs := []*netlink.Addr{}
	neighborIPs := []*netlink.Addr{}
	gateways := []*netlink.Addr{}

	ConnectionContext := nsc.Connection.GetContext()
	if ConnectionContext != nil {
		IpContext := ConnectionContext.GetIpContext()
		if IpContext != nil {
			localIP, err := netlink.ParseAddr(IpContext.SrcIpAddr)
			if err != nil {
				logrus.Errorf("Network Service Client: err parsing local IP: %v", err)
			}
			localIPs = []*netlink.Addr{localIP}
			neighborIP, err := netlink.ParseAddr(IpContext.DstIpAddr)
			if err != nil {
				logrus.Errorf("Network Service Client: err parsing neighbor IP: %v", err)
			}
			neighborIPs = []*netlink.Addr{neighborIP}
			if len(IpContext.ExtraPrefixes) > 0 {
				gateway, err := netlink.ParseAddr(IpContext.ExtraPrefixes[0])
				if err != nil {
					logrus.Errorf("Network Service Client: err parsing routes IP: %v", err)
				}
				gateways = []*netlink.Addr{gateway}
			}
		}
	}

	nsc.intf = networking.NewInterface(index, localIPs, neighborIPs)
	nsc.intf.InteraceType = networking.NSC
	nsc.intf.Gateways = gateways
}

func (nsc *NetworkServiceClient) advertiseInterfaceCreation() {
	if nsc.InterfaceMonitorSubscriber != nil {
		nsc.InterfaceMonitorSubscriber.InterfaceCreated(nsc.intf)
	}
}

func (nsc *NetworkServiceClient) advertiseInterfaceDeletion() {
	if nsc.InterfaceMonitorSubscriber != nil {
		nsc.InterfaceMonitorSubscriber.InterfaceDeleted(nsc.intf)
	}
}

func (nsc *NetworkServiceClient) prepareIpContext() *networkservice.IPContext {
	var ipContext *networkservice.IPContext
	if nsc.nscConnectionFactory != nil {
		var err error
		ipContext, err = nsc.nscConnectionFactory.NewNSCIPContext()
		if err != nil {
			logrus.Errorf("Network Service Client: err creating IP Context: %v", err)
			return nil
		}
		return ipContext
	}
	return nil
}

func (nsc *NetworkServiceClient) prepareRequest() *networkservice.NetworkServiceRequest {
	// TODO
	outgoingMechanism := &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: kernel.MECHANISM,
		Parameters: map[string]string{
			kernel.InterfaceNameKey: nsc.InterfaceName,
		},
	}
	nsc.Labels["forwarder"] = "forwarder-vpp"
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:                         nsc.Id,
			NetworkService:             nsc.NetworkServiceName,
			Labels:                     nsc.Labels,
			NetworkServiceEndpointName: nsc.NetworkServiceEndpointName,
			Payload:                    payload.Ethernet,
			Context: &networkservice.ConnectionContext{
				IpContext:    nsc.prepareIpContext(),
				ExtraContext: nsc.ExtraContext,
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			outgoingMechanism,
		},
	}
	return request
}

// NewnetworkServiceClient
func NewNetworkServiceClient(networkServiceName string, nsmgrClient NSMgrClient) *NetworkServiceClient {
	identifier := rand.Intn(100)
	id := fmt.Sprintf("%d", identifier)

	// TODO
	randomID := rand.Intn(1000)
	interfaceName := "nsc" + strconv.Itoa(randomID)

	networkServiceClient := &NetworkServiceClient{
		Id:                 id,
		NetworkServiceName: networkServiceName,
		nsmgrClient:        nsmgrClient,
		InterfaceName:      interfaceName,
		Labels:             make(map[string]string),
	}

	return networkServiceClient
}
