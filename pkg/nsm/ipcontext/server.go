package ipcontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/networking"
)

type ipcontextServer struct {
	ics ipContextSetter
}

// NewServer
func NewServer(ipContextSetter ipContextSetter) networkservice.NetworkServiceServer {
	return &ipcontextServer{
		ics: ipContextSetter,
	}
}

// Request
func (ics *ipcontextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	err := ics.ics.SetIPContext(request.Connection, networking.NSE)
	if err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

// Close
func (ics *ipcontextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO: free IPs
	return next.Server(ctx).Close(ctx, conn)
}
