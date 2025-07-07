/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2024-2025 OpenInfra Foundation Europe

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
	"time"

	"github.com/go-logr/logr"
	ipamAPI "github.com/nordix/meridio/api/ipam/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/logger"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/nordix/meridio/pkg/ipam/trench"
	"github.com/nordix/meridio/pkg/ipam/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const nodeUpdateDampingThreshold = 1 * time.Minute // update node prefix if last update older than threshold // TODO: configuration

type IpamServer struct {
	ctx    context.Context
	logger logr.Logger
	ipamAPI.UnimplementedIpamServer
	Trenches      map[ipamAPI.IPFamily]types.Trench
	PrefixLengths map[ipamAPI.IPFamily]*types.PrefixLengths
}

// NewIpam -
func NewServer(
	ctx context.Context,
	datastore string,
	trenchName string,
	nspConn *grpc.ClientConn,
	cidrs map[ipamAPI.IPFamily]string,
	prefixLengths map[ipamAPI.IPFamily]*types.PrefixLengths,
	garbageCollectionEnabled bool,
	garbageCollectionInterval time.Duration,
	garbageCollectionThreshold time.Duration) (ipamAPI.IpamServer, error) {
	is := &IpamServer{
		ctx:           ctx,
		logger:        logr.FromContextOrDiscard(ctx).WithValues("class", "IpamServer"),
		Trenches:      make(map[ipamAPI.IPFamily]types.Trench),
		PrefixLengths: prefixLengths,
	}
	store, err := sqlite.New(datastore)
	if err != nil {
		return nil, fmt.Errorf("failed creating new sqlite store (%s): %w", datastore, err)
	}
	if garbageCollectionEnabled {
		store.StartGarbageCollector(ctx, garbageCollectionInterval, garbageCollectionThreshold)
	}
	logStore := &logger.Store{
		Store: store,
	}

	trenchWatchers := []trench.TrenchWatcher{}
	for ipFamily, cidr := range cidrs {
		name := getTrenchName(trenchName, ipFamily)
		p := prefix.New(name, cidr, nil)
		newTrench, err := trench.New(ctx, p, logStore, is.PrefixLengths[ipFamily])
		if err != nil {
			return nil, fmt.Errorf("failed creating new trench prefix (%s): %w", p.GetName(), err)
		}
		is.Trenches[ipFamily] = newTrench
		trenchWatchers = append(trenchWatchers, newTrench)
	}
	configurationManagerClient := nspAPI.NewConfigurationManagerClient(nspConn)
	go trench.NewConduitWatcher(ctx, configurationManagerClient, trenchName, trenchWatchers)

	return is, nil
}

// Note: IMHO ideally all allocations should be treated as leases. Meaning they
// should be required to be renewed periodically instead of being allocated forever.
// Would the API be extended so that users could choose between lease or permanent
// allocation, it would make the cluster upgrade problematic due to the possible
// mix of old and new clients. Hence, IMHO the best approach for now is to let the
// server decide which prefixes should have their expirable attribute set. Currently,
// this translates to calls of node.Allocate() (excluding bridges) and conduit.GetNode().
// The information regarding the time scope is passed in context to avoid the need for
// changing all the API in between.
func (is *IpamServer) Allocate(ctx context.Context, child *ipamAPI.Child) (*ipamAPI.Prefix, error) {
	ctx = logr.NewContext(ctx, is.logger)
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
	if trench.GetName() != getTrenchName(child.GetSubnet().GetConduit().GetTrench().GetName(), child.GetSubnet().GetIpFamily()) {
		return nil, fmt.Errorf("no corresponding trench")
	}
	var conduit types.Conduit
	for {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("failed allocating (%s), ctx error: %w", child.GetName(), ctx.Err())
		}
		var err error
		conduit, err = trench.GetConduit(ctx, child.GetSubnet().GetConduit().GetName())
		if err != nil {
			return nil, fmt.Errorf("failed getting conduit (%s) while allocating (%s): %w", child.GetSubnet().GetConduit().GetName(), child.GetName(), err)
		}
		if conduit != nil {
			break
		}
	}
	getNodeCtx := sqlite.WithUpdateDamping(sqlite.WithExpirable(ctx), nodeUpdateDampingThreshold) // Applies damping to reduce frequent DB writes and marks node prefix as expirable
	node, err := conduit.GetNode(getNodeCtx, child.GetSubnet().GetNode())                         // Note: refreshes existing node prefix (i.e. the 'updatedAt' timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed getting node (%s) while allocating (%s): %w", child.GetSubnet().GetNode(), child.GetName(), err)
	}
	p, err := node.Allocate(sqlite.WithExpirable(ctx), child.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed allocating (%s): %w", child.GetName(), err)
	}
	ip, _, err := net.ParseCIDR(p.GetCidr())
	if err != nil {
		return nil, fmt.Errorf("failed ParseCIDR (%s) while allocating (%s): %w", p.GetCidr(), child.GetName(), err)
	}
	ret := &ipamAPI.Prefix{
		Address:      ip.String(),
		PrefixLength: int32(is.PrefixLengths[child.GetSubnet().GetIpFamily()].NodeLength),
	}
	return ret, nil
}

func (is *IpamServer) Release(ctx context.Context, child *ipamAPI.Child) (*emptypb.Empty, error) {
	ctx = logr.NewContext(ctx, is.logger)
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
	if trench.GetName() != getTrenchName(child.GetSubnet().GetConduit().GetTrench().GetName(), child.GetSubnet().GetIpFamily()) {
		return &emptypb.Empty{}, nil
	}
	conduit, err := trench.GetConduit(ctx, child.GetSubnet().GetConduit().GetName())
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed getting conduit (%s) while releasing (%s): %w", child.GetSubnet().GetNode(), child.GetName(), err)
	}
	if conduit == nil {
		return &emptypb.Empty{}, nil
	}
	// Note: Currently also refreshes existing node prefix (i.e. the 'updatedAt' timestamp).
	// Not sure node refresh is needed in case of Release, but errors must be avoided for
	// sure if node was reaped by a Garbage Collector logic.
	getNodeCtx := sqlite.WithUpdateDamping(sqlite.WithExpirable(ctx), nodeUpdateDampingThreshold)
	node, err := conduit.GetNode(getNodeCtx, child.GetSubnet().GetNode())
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed getting node (%s) while releasing (%s): %w", child.GetSubnet().GetNode(), child.GetName(), err)
	}
	if node == nil {
		return &emptypb.Empty{}, nil
	}
	err = node.Release(ctx, child.GetName())
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed releasing (%s): %w", child.GetName(), err)
	}
	return &emptypb.Empty{}, nil
}

func getTrenchName(trenchName string, ipFamily ipamAPI.IPFamily) string {
	return fmt.Sprintf("%s-%d", trenchName, ipFamily)
}
