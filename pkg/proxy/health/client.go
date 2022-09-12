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

package health

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/log"
	"google.golang.org/grpc"
)

type healthServiceClient struct {
	mu      sync.Mutex
	connIds map[string]struct{}
}

// NewClient -
// Creates nsm networkservice.NetworkServiceClient that reports
// health status for service 'EgressSvc'. Status is considered
// SERVING if at least 1 connection is available.
func NewClient() networkservice.NetworkServiceClient {
	return &healthServiceClient{
		connIds: make(map[string]struct{}),
	}
}

// Request -
func (h *healthServiceClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	id := request.Connection.Id
	resp, err := next.Client(ctx).Request(ctx, request, opts...)

	if err == nil {
		logger := log.FromContextOrGlobal(ctx)
		logger.V(2).Info("HealthServiceClient:Request", "id", id)
		h.mu.Lock()
		h.connIds[id] = struct{}{}
		health.SetServingStatus(ctx, health.EgressSvc, true)
		h.mu.Unlock()
	}

	return resp, err
}

// Close -
func (h *healthServiceClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContextOrGlobal(ctx)
	logger.V(2).Info("HealthServiceClient:Close", "id", conn.Id)
	h.mu.Lock()
	delete(h.connIds, conn.Id)
	if len(h.connIds) == 0 {
		logger.V(2).Info("HealthServiceClient:Close No conns left!")
		health.SetServingStatus(ctx, health.EgressSvc, false)
	}
	h.mu.Unlock()

	return next.Client(ctx).Close(ctx, conn, opts...)
}
