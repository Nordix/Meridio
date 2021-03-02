package ipam

import (
	"context"
	"fmt"
	"net"
	"strconv"

	ipamAPI "github.com/nordix/nvip/api/ipam"

	"github.com/golang/protobuf/ptypes/empty"
	goipam "github.com/metal-stack/go-ipam"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type IpamService struct {
	Listener net.Listener
	Server   *grpc.Server
	Port     int
	goIpam   goipam.Ipamer
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
func (is *IpamService) Allocate(ctx context.Context, SubnetRequest *ipamAPI.SubnetRequest) (*ipamAPI.Subnet, error) {
	subnetRequested := fmt.Sprintf("%s/%s", SubnetRequest.SubnetPool.Address, strconv.Itoa(int(SubnetRequest.SubnetPool.PrefixLength)))

	if _, ok := is.subnets[subnetRequested]; ok == false {
		is.subnets[subnetRequested] = struct{}{}
		_, err := is.goIpam.NewPrefix(subnetRequested)
		if err != nil {
			return nil, err
		}
	}

	child, err := is.goIpam.AcquireChildPrefix(subnetRequested, uint8(SubnetRequest.PrefixLength))
	if err != nil {
		return nil, err
	}

	return &ipamAPI.Subnet{
		Address:      child.Cidr,
		PrefixLength: SubnetRequest.PrefixLength,
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

	goIpam := goipam.New()

	s := grpc.NewServer()

	ipamService := &IpamService{
		Listener: lis,
		Server:   s,
		Port:     port,
		goIpam:   goIpam,
		subnets:  make(map[string]struct{}),
	}

	ipamAPI.RegisterIpamServiceServer(s, ipamService)

	return ipamService, nil
}
