package ipam

import (
	"context"
	"fmt"
	"net"
	"strconv"

	ipamAPI "github.com/nordix/meridio/api/ipam"
	"github.com/vishvananda/netlink"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type IpamService struct {
	Listener net.Listener
	Server   *grpc.Server
	Port     int
	ipam     *Ipam
	subnets  map[string]struct{}
}

// Start -
func (is *IpamService) Start() {
	logrus.Infof("IPAM Service: Start the service (port: %v)", is.Port)
	if err := is.Server.Serve(is.Listener); err != nil {
		logrus.Errorf("IPAM Service: failed to serve: %v", err)
	}
}

// Allocate -
func (is *IpamService) Allocate(ctx context.Context, subnetRequest *ipamAPI.SubnetRequest) (*ipamAPI.Subnet, error) {
	subnetRequestedCidr := fmt.Sprintf("%s/%s", subnetRequest.SubnetPool.Address, strconv.Itoa(int(subnetRequest.SubnetPool.PrefixLength)))
	subnetNetlink, err := netlink.ParseAddr(subnetRequestedCidr)
	if err != nil {
		return nil, err
	}

	subnet, err := is.ipam.AllocateSubnet(subnetNetlink, int(subnetRequest.PrefixLength))
	if err != nil {
		return nil, err
	}

	return &ipamAPI.Subnet{
		Address:      subnet.IP.String(),
		PrefixLength: subnetRequest.PrefixLength,
	}, nil
}

// Release -
func (is *IpamService) Release(ctx context.Context, subnetRelease *ipamAPI.SubnetRelease) (*empty.Empty, error) {
	return nil, nil
}

// NewIpam -
func NewIpamService(port int) (*IpamService, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		logrus.Errorf("IPAM Service: failed to listen: %v", err)
		return nil, err
	}

	ipam := NewIpam()

	s := grpc.NewServer()

	ipamService := &IpamService{
		Listener: lis,
		Server:   s,
		Port:     port,
		ipam:     ipam,
		subnets:  make(map[string]struct{}),
	}

	ipamAPI.RegisterIpamServiceServer(s, ipamService)

	return ipamService, nil
}
