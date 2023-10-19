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

package ipcontext

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/kernel"
	"google.golang.org/protobuf/types/known/emptypb"
)

type metricsServer struct {
	InterfaceMetrics *kernel.InterfaceMetrics
}

// NewServer
func NewServer(interfaceMetrics *kernel.InterfaceMetrics) networkservice.NetworkServiceServer {
	return &metricsServer{
		InterfaceMetrics: interfaceMetrics,
	}
}

// Request
func (ms *metricsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request == nil ||
		request.Connection == nil ||
		request.Connection.GetMechanism() == nil ||
		request.Connection.GetMechanism().GetParameters() == nil {
		return next.Server(ctx).Request(ctx, request)
	}
	interfaceName := request.Connection.GetMechanism().GetParameters()[common.InterfaceNameKey]
	ms.InterfaceMetrics.Register(interfaceName)
	return next.Server(ctx).Request(ctx, request)
}

// Close
func (ms *metricsServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	if conn == nil ||
		conn.GetMechanism() == nil ||
		conn.GetMechanism().GetParameters() == nil {
		return next.Server(ctx).Close(ctx, conn)
	}
	interfaceName := conn.GetMechanism().GetParameters()[common.InterfaceNameKey]
	ms.InterfaceMetrics.Unregister(interfaceName)
	return next.Server(ctx).Close(ctx, conn)
}
