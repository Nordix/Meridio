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

	ipamAPI "github.com/nordix/meridio/api/ipam"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/golang/protobuf/ptypes/empty"
)

type IpamService struct {
	ipam types.Ipam
}

// NewIpam -
func NewServer(datastore string) (ipamAPI.IpamServiceServer, error) {
	store, err := sqlite.NewStorage(datastore)
	if err != nil {
		return nil, err
	}
	im := NewWithStorage(store)

	ipamService := &IpamService{
		ipam: im,
	}

	return ipamService, nil
}

// Allocate -
func (is *IpamService) Allocate(ctx context.Context, subnetRequest *ipamAPI.SubnetRequest) (*ipamAPI.Subnet, error) {
	subnetRequestedCidr := fmt.Sprintf("%s/%d", subnetRequest.SubnetPool.Address, int(subnetRequest.SubnetPool.PrefixLength))

	logrus.Infof("Allocate: %v", subnetRequest)
	subnet, err := is.ipam.AllocateSubnet(ctx, subnetRequestedCidr, int(subnetRequest.PrefixLength))
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
