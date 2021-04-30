package ipcontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type ipcontextClient struct {
	ics ipContextSetter
}

// NewClient
func NewClient(ipContextSetter ipContextSetter) networkservice.NetworkServiceClient {
	return &ipcontextClient{
		ics: ipContextSetter,
	}
}

// Request
func (icc *ipcontextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	err := icc.ics.SetIPContext(request.Connection, networking.NSC)
	if err != nil {
		return nil, err
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

// Close
func (icc *ipcontextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// TODO: free IPs
	return next.Client(ctx).Close(ctx, conn, opts...)
}
