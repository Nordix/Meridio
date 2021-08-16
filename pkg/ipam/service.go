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

	subnet, err := is.ipam.AllocateSubnet(subnetRequestedCidr, int(subnetRequest.PrefixLength))
	if err != nil {
		return nil, err
	}

	subnetAddr, err := netlink.ParseAddr(subnet)
	if err != nil {
		return nil, err
	}

	return &ipamAPI.Subnet{
		Address:      subnetAddr.IP.String(),
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
