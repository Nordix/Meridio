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

type Ipam struct {
	Listener net.Listener
	Server   *grpc.Server
	Port     int
	goIpam   goipam.Ipamer
	subnets  map[string]struct{}
}

// Start -
func (ipam *Ipam) Start() {
	logrus.Infof("IPAM Service: Start the service (port: %v)", ipam.Port)
	if err := ipam.Server.Serve(ipam.Listener); err != nil {
		logrus.Errorf("IPAM Service: failed to serve: %v", err)
	}
}

// Allocate -
func (ipam *Ipam) Allocate(ctx context.Context, SubnetRequest *ipamAPI.SubnetRequest) (*ipamAPI.Subnet, error) {
	subnetRequested := fmt.Sprintf("%s/%s", SubnetRequest.Subnet.Address, strconv.Itoa(int(SubnetRequest.Subnet.PrefixLength)))

	if _, ok := ipam.subnets[subnetRequested]; ok == false {
		ipam.subnets[subnetRequested] = struct{}{}
		_, err := ipam.goIpam.NewPrefix(subnetRequested)
		if err != nil {
			return nil, err
		}
	}

	child, err := ipam.goIpam.AcquireChildPrefix(subnetRequested, uint8(SubnetRequest.PrefixLength))
	if err != nil {
		return nil, err
	}

	return &ipamAPI.Subnet{
		Address:      child.Cidr,
		PrefixLength: SubnetRequest.PrefixLength,
	}, nil
}

// Release -
func (ipam *Ipam) Release(ctx context.Context, subnetRelease *ipamAPI.SubnetRelease) (*empty.Empty, error) {
	return nil, nil
}

// NewIpam -
func NewIpam(port int) (*Ipam, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		logrus.Errorf("IPAM Service: failed to listen: %v", err)
		return nil, err
	}

	goIpam := goipam.New()

	s := grpc.NewServer()

	ipam := &Ipam{
		Listener: lis,
		Server:   s,
		Port:     port,
		goIpam:   goIpam,
		subnets:  make(map[string]struct{}),
	}

	ipamAPI.RegisterIpamServiceServer(s, ipam)

	return ipam, nil
}
