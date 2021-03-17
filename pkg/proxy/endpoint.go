package proxy

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/sirupsen/logrus"
)

// ProxyEndpoint -
type ProxyEndpoint struct {
	nseConnectionFactory endpoint.NSEConnectionFactory
}

// NewProxyEndpoint -
func NewProxyEndpoint(nseConnectionFactory endpoint.NSEConnectionFactory) *ProxyEndpoint {
	return &ProxyEndpoint{
		nseConnectionFactory: nseConnectionFactory,
	}
}

// Request -
func (pe *ProxyEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	ipContext, err := pe.nseConnectionFactory.NewNSEIPContext()
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err creating new IP context: %v", err)
	}
	request.GetConnection().GetContext().IpContext = ipContext

	return next.Server(ctx).Request(ctx, request)
}

// Close -
func (pe *ProxyEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
