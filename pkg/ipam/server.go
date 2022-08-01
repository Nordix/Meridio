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

	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

type IpamServer struct {
	ipamAPI.UnimplementedIpamServer
	Trenches      map[ipamAPI.IPFamily]types.Trench
	PrefixLengths map[ipamAPI.IPFamily]*types.PrefixLengths
	Logger        log.Logger
}

// NewIpam -
func NewServer(trenches map[ipamAPI.IPFamily]types.Trench,
	prefixLengths map[ipamAPI.IPFamily]*types.PrefixLengths,
	logger log.Logger) (ipamAPI.IpamServer, error) {
	is := &IpamServer{
		Trenches:      trenches,
		PrefixLengths: prefixLengths,
		Logger:        logger,
	}

	return is, nil
}

func (is *IpamServer) Allocate(ctx context.Context, child *ipamAPI.Child) (*ipamAPI.Prefix, error) {
	ctx = log.WithLogger(ctx, is.Logger)
	is.Logger.Info("Allocate: %v", child)
	trench, exists := is.Trenches[child.GetSubnet().GetIpFamily()]
	if !exists {
		return nil, fmt.Errorf("cannot allocate in this ip family")
	}
	if child.GetSubnet().GetConduit() == nil {
		return nil, fmt.Errorf("conduit cannot be nil")
	}
	if child.GetSubnet().GetConduit().GetTrench() == nil {
		return nil, fmt.Errorf("trench cannot be nil")
	}
	if trench.GetName() != GetTrenchName(child.GetSubnet().GetConduit().GetTrench().GetName(), child.GetSubnet().GetIpFamily()) {
		return nil, fmt.Errorf("no corresponding trench")
	}
	var conduit types.Conduit
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var err error
		conduit, err = trench.GetConduit(ctx, child.GetSubnet().GetConduit().GetName())
		if err != nil {
			return nil, err
		}
		if conduit != nil {
			break
		}
	}
	node, err := conduit.GetNode(ctx, child.GetSubnet().GetNode())
	if err != nil {
		return nil, err
	}
	p, err := node.Allocate(ctx, child.GetName())
	if err != nil {
		return nil, err
	}
	ip, _, err := net.ParseCIDR(p.GetCidr())
	if err != nil {
		return nil, err
	}
	return &ipamAPI.Prefix{
		Address:      ip.String(),
		PrefixLength: int32(is.PrefixLengths[child.GetSubnet().GetIpFamily()].NodeLength),
	}, nil
}

func (is *IpamServer) Release(ctx context.Context, child *ipamAPI.Child) (*emptypb.Empty, error) {
	ctx = log.WithLogger(ctx, is.Logger)
	is.Logger.Info("Release: %v", child)
	trench, exists := is.Trenches[child.GetSubnet().GetIpFamily()]
	if !exists {
		return &emptypb.Empty{}, nil
	}
	if child.GetSubnet().GetConduit() == nil {
		return &emptypb.Empty{}, nil
	}
	if child.GetSubnet().GetConduit().GetTrench() == nil {
		return &emptypb.Empty{}, nil
	}
	if trench.GetName() != GetTrenchName(child.GetSubnet().GetConduit().GetTrench().GetName(), child.GetSubnet().GetIpFamily()) {
		return &emptypb.Empty{}, nil
	}
	conduit, err := trench.GetConduit(ctx, child.GetSubnet().GetConduit().GetName())
	if err != nil {
		return &emptypb.Empty{}, err
	}
	if conduit == nil {
		return &emptypb.Empty{}, nil
	}
	node, err := conduit.GetNode(ctx, child.GetSubnet().GetNode())
	if err != nil {
		return &emptypb.Empty{}, err
	}
	if node == nil {
		return &emptypb.Empty{}, nil
	}
	return &emptypb.Empty{}, node.Release(ctx, child.GetName())
}

func GetTrenchName(trenchName string, ipFamily ipamAPI.IPFamily) string {
	return fmt.Sprintf("%s-%d", trenchName, ipFamily)
}
