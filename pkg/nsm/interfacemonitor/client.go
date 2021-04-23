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

// NewClient -
func NewClient(interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber, netUtils networking.Utils) networkservice.NetworkServiceClient {
	return &interfaceMonitorClient{
		NewInterfaceMonitor(interfaceMonitorSubscriber, netUtils),
	}
}

func (inc *interfaceMonitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return conn, err
	}
	inc.ConnectionRequested(&connection{conn}, networking.NSC)
	return conn, err
}

func (inc *interfaceMonitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if conn != nil {
		inc.ConnectionClosed(&connection{conn}, networking.NSC)
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
