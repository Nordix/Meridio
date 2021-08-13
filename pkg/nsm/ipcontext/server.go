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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/networking"
)

type ipcontextServer struct {
	ics ipContextSetter
}

// NewServer
func NewServer(ipContextSetter ipContextSetter) networkservice.NetworkServiceServer {
	return &ipcontextServer{
		ics: ipContextSetter,
	}
}

// Request
func (ics *ipcontextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	err := ics.ics.SetIPContext(request.Connection, networking.NSE)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

// Close
func (ics *ipcontextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO: free IPs
	return next.Server(ctx).Close(ctx, conn)
}
