/*
Copyright (c) 2022 Nordix Foundation

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

package fullmeshtracker

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type fullMeshTrackerClient struct {
	m sync.Map // key: connId, value: NSE-Name
}

// NewClient implements NetworkServiceClient to track the connections and the NSE name requested
// in order to prevent NSM from selecting the NSE. In case a NSC wants to connect to a specific NSE
// and that NSE has a problem (crash...), NSM will remove the NSE name from the connection to select a new one.
// This chain element will remember the NSE name and add it again, so the connection will always be to the
// originally selected NSE.
func NewClient() networkservice.NetworkServiceClient {
	return &fullMeshTrackerClient{}
}

// Request stores the NSE name if not previously saved. If it is previously saved, then it loads it to
// add it to the connection.
func (fmtc *fullMeshTrackerClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	nseName, exists := fmtc.m.Load(request.Connection.Id)
	if exists {
		request.Connection.NetworkServiceEndpointName = nseName.(string)
	} else {
		fmtc.m.Store(request.Connection.Id, request.Connection.NetworkServiceEndpointName)
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

// Close -
// TODO: remove NSE from the map when they are removed (real close)
func (fmtc *fullMeshTrackerClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn, opts...)
	return &emptypb.Empty{}, err
}
