package interfacename

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type interfaceNameClient struct {
	*interfaceNameSetter
}

// NewClient -
func NewClient(prefix string, generator NameGenerator) networkservice.NetworkServiceClient {
	return &interfaceNameClient{
		NewInterfaceNameSetter(prefix, generator, MAX_INTERFACE_NAME_LENGTH),
	}
}

func (inc *interfaceNameClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// TODO: check if interface name already exists
	inc.SetInterfaceName(request)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (inc *interfaceNameClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
