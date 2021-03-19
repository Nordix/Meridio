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
		ip := request.GetConnection().GetContext().GetIpContext().GetSrcIpAddr()
		context := request.GetConnection().GetContext().GetExtraContext()
		err := nspe.networkServicePlateformClient.Register(ip, context)
		logrus.Infof("NSPEndpoint: Register ip: ", ip, err)
	}
	return next.Server(ctx).Request(ctx, request)
}

func (nspe *NSPEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logrus.Infof("NSPEndpoint: Close")
	if conn.GetContext() != nil && conn.GetContext().GetIpContext() != nil {
		ip := conn.GetContext().GetIpContext().GetSrcIpAddr()
		err := nspe.networkServicePlateformClient.Unregister(ip)
		logrus.Infof("NSPEndpoint: Unregister ip: ", ip, err)
	}
	return next.Server(ctx).Close(ctx, conn)
}
