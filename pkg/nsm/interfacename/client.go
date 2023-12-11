/*
Copyright (c) 2021-2023 Nordix Foundation

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

package interfacename

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type interfaceNameClient struct {
	*interfaceNameSetter
}

// NewClient implements NetworkServiceClient to generate and add the interface name in the
// mechanism and mechanism preferences of the requests
func NewClient(prefix string, generator NameGenerator) networkservice.NetworkServiceClient {
	return &interfaceNameClient{
		newInterfaceNameSetter(prefix, generator, MAX_INTERFACE_NAME_LENGTH),
	}
}

// Request sets the value for the common.InterfaceNameKey key in the parameters of the mechanism
// A non-nil error is returned if the name generation fails or if a next element in the chain returns a non-nil error
// It implements NetworkServiceClient for the interfacename package
func (inc *interfaceNameClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	inc.SetInterfaceName(request)
	connection, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		inc.UnsetInterfaceName(request)
		return connection, fmt.Errorf("failed to request (%s) connection to NSM (interfaceNameClient): %w", request.String(), err)
	}

	return connection, nil
}

// Close it does nothing except calling the next Close in the chain
// A non-nil error if a next element in the chain returns a non-nil error
// It implements NetworkServiceClient for the interfacename package
func (inc *interfaceNameClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	inc.UnsetInterfaceName(conn)
	empty, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return empty, fmt.Errorf("failed to close (%s) connection from NSM (interfaceNameClient): %w", conn.String(), err)
	}

	return empty, nil
}
