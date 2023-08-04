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
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (c *defaultExpirationNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *defaultExpirationNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
