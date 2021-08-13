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
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/networking"
)

type interfaceMonitorEndpoint struct {
	*interfaceMonitor
}

// NewServer implements NetworkServiceServer to advertise the interfaceMonitorSubscriber when a
// NSM interface has been created / removed in the pod
// A networking.InterfaceMonitor is required to get events on interface creation / deletion on the pod
// A networkingUtils (e.g. kernel implementation) is needed in order to check if an interface is
// existing in the pod, and to create the interface to return to the interfaceMonitorSubscriber
func NewServer(interfaceMonitor networking.InterfaceMonitor, interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networkingUtils) networkservice.NetworkServiceServer {
	return &interfaceMonitorEndpoint{
		newInterfaceMonitor(interfaceMonitor, interfaceMonitorSubscriber, netUtils),
	}
}

// Request will call the InterfaceCreated function in the interfaceMonitorSubscriber when
// the network interface requested will be created in the pod
func (ime *interfaceMonitorEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request != nil && request.GetConnection() != nil {
		ime.ConnectionRequested(&connection{request.GetConnection()}, networking.NSE)
	}
	return next.Server(ctx).Request(ctx, request)
}

// Close will call the InterfaceDeleted function in the interfaceMonitorSubscriber
func (ime *interfaceMonitorEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn != nil {
		ime.ConnectionClosed(&connection{conn}, networking.NSE)
	}
	return next.Server(ctx).Close(ctx, conn)
}
