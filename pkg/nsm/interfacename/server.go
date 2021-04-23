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

// NewInterfaceNameEndpoint -
func NewServer(prefix string, generator NameGenerator) networkservice.NetworkServiceServer {
	return &interfaceNameServer{
		NewInterfaceNameSetter(prefix, generator, MAX_INTERFACE_NAME_LENGTH),
	}
}

// Request -
func (ine *interfaceNameServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// TODO: check if interface name already exists
	ine.SetInterfaceName(request)
	return next.Server(ctx).Request(ctx, request)
}

// Close -
func (ine *interfaceNameServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
