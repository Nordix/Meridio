package nsp

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"google.golang.org/grpc"
)

type NetworkServicePlateformClient struct {
	networkServicePlateformClient nspAPI.NetworkServicePlateformServiceClient
}

func (nspc *NetworkServicePlateformClient) Register(ip string, targetContext map[string]string) error {
	target := &nspAPI.Target{
		Ip:      ip,
		Context: targetContext,
	}
	_, err := nspc.networkServicePlateformClient.Register(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) Unregister(ip string) error {
	target := &nspAPI.Target{
		Ip: ip,
	}
	_, err := nspc.networkServicePlateformClient.Unregister(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) Monitor() (nspAPI.NetworkServicePlateformService_MonitorClient, error) {
	return nspc.networkServicePlateformClient.Monitor(context.Background(), &empty.Empty{})
}

func (nspc *NetworkServicePlateformClient) GetTargets() error {
	return nil
}

func (nspc *NetworkServicePlateformClient) connect(ipamServiceIPPort string) error {
	conn, err := grpc.Dial(ipamServiceIPPort, grpc.WithInsecure())
	if err != nil {
		return nil
	}

	nspc.networkServicePlateformClient = nspAPI.NewNetworkServicePlateformServiceClient(conn)
	return nil
}

func NewNetworkServicePlateformClient(serviceIPPort string) (*NetworkServicePlateformClient, error) {
	networkServicePlateformClient := &NetworkServicePlateformClient{}
	err := networkServicePlateformClient.connect(serviceIPPort)
	if err != nil {
		return nil, err
	}
	return networkServicePlateformClient, nil
}
