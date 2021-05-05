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

package interfacename

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type interfaceNameServer struct {
	*interfaceNameSetter
}

// NewServer implements NetworkServiceServer to generate and add the interface name in the
// mechanism and mechanism preferences of the requests
func NewServer(prefix string, generator NameGenerator) networkservice.NetworkServiceServer {
	return &interfaceNameServer{
		newInterfaceNameSetter(prefix, generator, MAX_INTERFACE_NAME_LENGTH),
	}
}

// Request sets the value for the common.InterfaceNameKey key in the parameters of the mechanism
// A non-nil error is returned if the name generation fails, if a next element in the chain returns a non-nil error
// It implements NetworkServiceServer for the interfacename package
func (ine *interfaceNameServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// TODO: check if interface name already exists
	ine.SetInterfaceName(request)
	return next.Server(ctx).Request(ctx, request)
}

// Close it does nothing except calling the next Close in the chain
// A non-nil error if a next element in the chain returns a non-nil error
// It implements NetworkServiceServer for the interfacename package
func (ine *interfaceNameServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
