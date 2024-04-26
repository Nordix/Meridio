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

package ipcontext

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ipcontextServer struct {
	ics            ipContextSetter
	ipReleaseDelay time.Duration
}

// NewServer
func NewServer(ipContextSetter ipContextSetter, ipReleaseDelay time.Duration) networkservice.NetworkServiceServer {
	return &ipcontextServer{
		ics:            ipContextSetter,
		ipReleaseDelay: ipReleaseDelay,
	}
}

// Request
func (ics *ipcontextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	err := ics.ics.SetIPContext(ctx, request.Connection, networking.NSE)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

// Close
func (ics *ipcontextServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	// Note: In case of TAPA connections, IPs are identified by the TAPA path id, which does
	// not contain a random part, thus IPs can be recovered.
	err := ics.ics.UnsetIPContext(ctx, conn, networking.NSE, ics.ipReleaseDelay)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Close(ctx, conn)
}
