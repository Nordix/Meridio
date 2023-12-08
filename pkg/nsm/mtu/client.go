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

package mtu

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// mtuClient adds proposed MTU to the Request. To be used by NSC.
// Won't take effect, if NSM cannot fit the proposed MTU considering
// the chosen mechanism and the underlying network infrastructure.
type mtuClient struct {
	mtu uint32
}

func NewMtuClient(mtu uint32) networkservice.NetworkServiceClient {
	return &mtuClient{
		mtu: mtu,
	}
}

func (m *mtuClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	request.GetConnection().GetContext().MTU = m.mtu

	connection, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return connection, fmt.Errorf("failed to request (%s) connection to NSM (mtuClient): %w", request.String(), err)
	}

	return connection, nil
}

func (m *mtuClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	empty, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return empty, fmt.Errorf("failed to close (%s) connection from NSM (mtuClient): %w", conn.String(), err)
	}

	return empty, err
}
