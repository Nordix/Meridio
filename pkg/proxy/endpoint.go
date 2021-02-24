package proxy

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type ProxyEndpoint struct {
}

func NewProxyEndpoint() networkservice.NetworkServiceServer {
	return &ProxyEndpoint{}
}

func (pe *ProxyEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	localIP, err := netlink.ParseAddr(request.GetConnection().GetContext().GetIpContext().DstIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing local IP: %v", err)
	}
	neighborIP, err := netlink.ParseAddr(request.GetConnection().GetContext().GetIpContext().SrcIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing neighbor IP: %v", err)
	}

	interfaceName := request.GetConnection().Mechanism.Parameters["name"]
	localIPs := []*netlink.Addr{localIP}
	neighborIPs := []*netlink.Addr{neighborIP}

	log.FromContext(ctx).Infof("ProxyEndpoint: (Request) interfaceName: %+v", interfaceName)
	log.FromContext(ctx).Infof("ProxyEndpoint: (Request) localIPs: %+v", localIPs)
	log.FromContext(ctx).Infof("ProxyEndpoint: (Request) neighborIPs: %+v", neighborIPs)

	return next.Server(ctx).Request(ctx, request)
}

func (pe *ProxyEndpoint) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {

	localIP, err := netlink.ParseAddr(conn.GetContext().GetIpContext().DstIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing local IP: %v", err)
	}
	neighborIP, err := netlink.ParseAddr(conn.GetContext().GetIpContext().SrcIpAddr)
	if err != nil {
		logrus.Errorf("ProxyEndpoint: err parsing neighbor IP: %v", err)
	}

	interfaceName := conn.Mechanism.Parameters["name"]
	localIPs := []*netlink.Addr{localIP}
	neighborIPs := []*netlink.Addr{neighborIP}

	log.FromContext(ctx).Infof("ProxyEndpoint: (Close) interfaceName: %+v", interfaceName)
	log.FromContext(ctx).Infof("ProxyEndpoint: (Close) localIPs: %+v", localIPs)
	log.FromContext(ctx).Infof("ProxyEndpoint: (Close) neighborIPs: %+v", neighborIPs)

	return next.Server(ctx).Close(ctx, conn)
}
