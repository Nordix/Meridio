package interfacemonitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/networking"
)

// InterfaceMonitorEndpoint -
type interfaceMonitorEndpoint struct {
	*interfaceMonitor
}

// NewInterfaceMonitorEndpoint -
func NewServer(interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networking.Utils) *interfaceMonitorEndpoint {
	return &interfaceMonitorEndpoint{
		NewInterfaceMonitor(interfaceMonitorSubscriber, netUtils),
	}
}

// Request -
func (ime *interfaceMonitorEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request != nil && request.GetConnection() != nil {
		ime.ConnectionRequested(&connection{request.GetConnection()}, networking.NSE)
	}
	return next.Server(ctx).Request(ctx, request)
}

// Close -
func (ime *interfaceMonitorEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn != nil {
		ime.ConnectionClosed(&connection{conn}, networking.NSE)
	}
	return next.Server(ctx).Close(ctx, conn)
}
