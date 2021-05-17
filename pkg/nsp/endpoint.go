package nsp

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/sirupsen/logrus"
)

type NSPEndpoint struct {
	networkServicePlateformClient *NetworkServicePlateformClient
}

func NewNSPEndpoint(nspService string) *NSPEndpoint {
	nspClient, _ := NewNetworkServicePlateformClient(nspService)
	return &NSPEndpoint{
		networkServicePlateformClient: nspClient,
	}
}

func (nspe *NSPEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logrus.Infof("NSPEndpoint: Request")
	if request.GetConnection().GetContext() != nil && request.GetConnection().GetContext().GetIpContext() != nil {
		ips := request.GetConnection().GetContext().GetIpContext().GetSrcIpAddrs()
		context := request.GetConnection().GetContext().GetExtraContext()
		err := nspe.networkServicePlateformClient.Register(ips, context)
		logrus.Infof("NSPEndpoint: Register ip: %v %v", ips, err)
	}
	return next.Server(ctx).Request(ctx, request)
}

func (nspe *NSPEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logrus.Infof("NSPEndpoint: Close")
	if conn.GetContext() != nil && conn.GetContext().GetIpContext() != nil {
		ips := conn.GetContext().GetIpContext().GetSrcIpAddrs()
		err := nspe.networkServicePlateformClient.Unregister(ips)
		logrus.Infof("NSPEndpoint: Unregister ip: %v %v", ips, err)
	}
	return next.Server(ctx).Close(ctx, conn)
}
