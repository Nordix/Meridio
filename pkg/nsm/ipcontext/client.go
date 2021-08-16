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
	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type ipcontextClient struct {
	ics ipContextSetter
}

// NewClient
func NewClient(ipContextSetter ipContextSetter) networkservice.NetworkServiceClient {
	return &ipcontextClient{
		ics: ipContextSetter,
	}
}

// Request
func (icc *ipcontextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	err := icc.ics.SetIPContext(request.Connection, networking.NSC)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

// Close
func (icc *ipcontextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// TODO: free IPs
	return next.Client(ctx).Close(ctx, conn, opts...)
}
