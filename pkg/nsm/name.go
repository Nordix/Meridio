package nsm

import (
	"context"
	"math/rand"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// InterfaceMonitorEndpoint -
type InterfaceNameEndpoint struct {
}

// NewInterfaceNameEndpoint -
func NewInterfaceNameEndpoint() *InterfaceNameEndpoint {
	return &InterfaceNameEndpoint{}
}

// Request -
func (ine *InterfaceNameEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetMechanism().GetParameters() == nil {
		request.GetConnection().GetMechanism().Parameters = make(map[string]string)
	}

	// TODO name generation
	randomID := rand.Intn(1000)
	interfaceName := "nse" + strconv.Itoa(randomID)
	request.GetConnection().GetMechanism().GetParameters()[kernel.InterfaceNameKey] = interfaceName

	return next.Server(ctx).Request(ctx, request)
}

// Close -
func (ine *InterfaceNameEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
