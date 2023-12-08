/*
Copyright (c) 2023 Nordix Foundation

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

package expirationtime

import (
	"context"
	"fmt"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type defaultExpirationNSEClient struct {
	defaultLifetime time.Duration
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will set
// NSE expiration time based on defaultLifetime during registration if no expiration time has been set yet.
func NewNetworkServiceEndpointRegistryClient(defaultLifetime time.Duration) registry.NetworkServiceEndpointRegistryClient {
	return &defaultExpirationNSEClient{
		defaultLifetime: defaultLifetime,
	}
}

func (c *defaultExpirationNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if nse.GetExpirationTime() == nil {
		timeClock := clock.FromContext(ctx)
		expirationTime := timeClock.Now().Add(c.defaultLifetime).Local()
		nse.ExpirationTime = timestamppb.New(expirationTime)
	}

	nse, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err != nil {
		return nse, fmt.Errorf("failed to register network service endpoint (%s) to NSM (defaultExpirationNSEClient): %w", nse.String(), err)
	}

	return nse, nil
}

func (c *defaultExpirationNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	findClient, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
	if err != nil {
		return findClient, fmt.Errorf("failed to fint network service endpoint (%s) in NSM (defaultExpirationNSEClient): %w", query.String(), err)
	}

	return findClient, nil
}

func (c *defaultExpirationNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	empty, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
	if err != nil {
		return empty, fmt.Errorf("failed to unregister network service endpoint (%s) to NSM (defaultExpirationNSEClient): %w", nse.String(), err)
	}

	return empty, nil
}
