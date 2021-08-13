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

package interfacemonitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type interfaceMonitorClient struct {
	*interfaceMonitor
}

// NewClient implements NetworkServiceClient to advertise the interfaceMonitorSubscriber when a
// NSM interface has been created / removed in the pod
// A networking.InterfaceMonitor is required to get events on interface creation / deletion on the pod
// A networkingUtils (e.g. kernel implementation) is needed in order to check if an interface is
// existing in the pod, and to create the interface to return to the interfaceMonitorSubscriber
func NewClient(interfaceMonitor networking.InterfaceMonitor, interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networkingUtils) networkservice.NetworkServiceClient {
	return &interfaceMonitorClient{
		newInterfaceMonitor(interfaceMonitor, interfaceMonitorSubscriber, netUtils),
	}
}

// Request will call the InterfaceCreated function in the interfaceMonitorSubscriber when
// the network interface requested will be created in the pod
func (inc *interfaceMonitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return conn, err
	}
	inc.ConnectionRequested(&connection{conn}, networking.NSC)
	return conn, err
}

// Close will call the InterfaceDeleted function in the interfaceMonitorSubscriber
func (inc *interfaceMonitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if conn != nil {
		inc.ConnectionClosed(&connection{conn}, networking.NSC)
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
