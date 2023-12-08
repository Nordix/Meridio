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
	"google.golang.org/protobuf/types/known/emptypb"
)

// mtuServer adds proposed MTU to the Request. To be used by NSE.
// Won't take effect, if NSM cannot fit the proposed MTU considering
// the chosen mechanism and the underlying network infrastructure.
type mtuServer struct {
	mtu uint32
}

func NewMtuServer(mtu uint32) networkservice.NetworkServiceServer {
	return &mtuServer{
		mtu: mtu,
	}
}

func (m *mtuServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	request.GetConnection().GetContext().MTU = m.mtu

	connection, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return connection, fmt.Errorf("failed to request (%s) connection to NSM (mtuServer): %w", request.String(), err)
	}

	return connection, err
}

func (m *mtuServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	empty, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return empty, fmt.Errorf("failed to close (%s) connection from NSM (mtuServer): %w", conn.String(), err)
	}

	return empty, nil
}
