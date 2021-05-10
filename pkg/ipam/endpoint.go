package ipam

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	goipam "github.com/metal-stack/go-ipam"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	ipamAPI "github.com/nordix/meridio/api/ipam"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type IpamEndpoint struct {
	ipamServiceClient    ipamAPI.IpamServiceClient
	networkServiceSubnet *ipamAPI.Subnet
	prefixLengthRequest  int
}

func NewIpamEndpoint(network *net.IPNet, ipamServiceIPPort string) *IpamEndpoint {
	conn, err := grpc.Dial(ipamServiceIPPort, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("IpamEndpoint: failed to connect to %v: %v", ipamServiceIPPort, err)
	}

	ipamServiceClient := ipamAPI.NewIpamServiceClient(conn)

	prefixLength, _ := network.Mask.Size()

	networkServiceSubnet := &ipamAPI.Subnet{
		Address:      network.IP.String(),
		PrefixLength: int32(prefixLength),
	}

	return &IpamEndpoint{
		ipamServiceClient:    ipamServiceClient,
		networkServiceSubnet: networkServiceSubnet,
		prefixLengthRequest:  30, // TODO IPV6
	}
}

func (ie *IpamEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	subnetRequest := &ipamAPI.SubnetRequest{
		SubnetPool:   ie.networkServiceSubnet,
		PrefixLength: int32(ie.prefixLengthRequest),
	}

	allocatedSubnet, err := ie.ipamServiceClient.Allocate(context.Background(), subnetRequest)
	allocatedSubnetCIDR := fmt.Sprintf("%s/%d", allocatedSubnet.Address, allocatedSubnet.PrefixLength)
	if err != nil {
		logrus.Errorf("IpamEndpoint: err to allocate new subnet: %v", err)
	}
	logrus.Infof("IpamEndpoint: AllocatedSubnet: %v", allocatedSubnet)

	connection := request.GetConnection()
	if connection.GetContext() == nil {
		connection.Context = &networkservice.ConnectionContext{}
	}
	if connection.GetContext().GetIpContext() == nil {
		connection.GetContext().IpContext = &networkservice.IPContext{}
	}
	ipContext := connection.GetContext().GetIpContext()

	goIpam := goipam.New()
	_, err = goIpam.NewPrefix(allocatedSubnetCIDR)
	if err != nil {
		logrus.Errorf("IpamEndpoint: err (goIpam) NewPrefix: %v", err)
	}
	srcIP, err := goIpam.AcquireIP(allocatedSubnetCIDR)
	if err != nil {
		logrus.Errorf("IpamEndpoint: err (goIpam) Acquire srcIP: %v", err)
	}
	dstIP, err := goIpam.AcquireIP(allocatedSubnetCIDR)
	if err != nil {
		logrus.Errorf("IpamEndpoint: err (goIpam) Acquire dstIP: %v", err)
	}

	sourceIP := fmt.Sprintf("%s/%d", srcIP.IP.String(), ie.prefixLengthRequest)
	destinationIP := fmt.Sprintf("%s/%d", dstIP.IP.String(), ie.prefixLengthRequest)

	ipContext.SrcIpAddrs = []string{sourceIP}
	ipContext.DstIpAddrs = []string{destinationIP}

	return next.Server(ctx).Request(ctx, request)
}

func (ie *IpamEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO free
	return next.Server(ctx).Close(ctx, conn)
}
