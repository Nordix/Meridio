/*
Copyright (c) 2021 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nsp

import (
	"context"

	nspAPI "github.com/nordix/meridio/api/nsp"
	"google.golang.org/grpc"
)

type NetworkServicePlateformClient struct {
	conn                          *grpc.ClientConn
	networkServicePlateformClient nspAPI.NetworkServicePlateformServiceClient
}

const (
	EnabledStatus  nspAPI.Target_Status = nspAPI.Target_Enabled
	DisabledStatus nspAPI.Target_Status = nspAPI.Target_Disabled
)

func (nspc *NetworkServicePlateformClient) Register(ips []string, targetContext map[string]string) error {
	target := &nspAPI.Target{
		Ips:     ips,
		Context: targetContext,
	}
	_, err := nspc.networkServicePlateformClient.Register(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) RegisterWithType(t nspAPI.Target_Type, ips []string, targetContext map[string]string) error {
	target := &nspAPI.Target{
		Ips:     ips,
		Type:    t,
		Context: targetContext,
	}
	_, err := nspc.networkServicePlateformClient.Register(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) Unregister(ips []string) error {
	target := &nspAPI.Target{
		Ips: ips,
	}
	_, err := nspc.networkServicePlateformClient.Unregister(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) UnregisterWithType(t nspAPI.Target_Type, ips []string) error {
	target := &nspAPI.Target{
		Ips:  ips,
		Type: t,
	}
	_, err := nspc.networkServicePlateformClient.Unregister(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) UnregisterWithContext(t nspAPI.Target_Type, ips []string, targetContext map[string]string) error {
	target := &nspAPI.Target{
		Ips:     ips,
		Type:    t,
		Context: targetContext,
	}
	_, err := nspc.networkServicePlateformClient.Unregister(context.Background(), target)
	return err
}

func (nspc *NetworkServicePlateformClient) Monitor() (nspAPI.NetworkServicePlateformService_MonitorClient, error) {
	targetType := &nspAPI.TargetType{
		Type: nspAPI.Target_DEFAULT,
	}
	return nspc.networkServicePlateformClient.Monitor(context.Background(), targetType)
}

func (nspc *NetworkServicePlateformClient) MonitorType(t nspAPI.Target_Type) (nspAPI.NetworkServicePlateformService_MonitorClient, error) {
	targetType := &nspAPI.TargetType{
		Type: t,
	}
	return nspc.networkServicePlateformClient.Monitor(context.Background(), targetType)
}

func (nspc *NetworkServicePlateformClient) GetTargets() ([]*nspAPI.Target, error) {
	targetType := &nspAPI.TargetType{}
	GetTargetsResponse, err := nspc.networkServicePlateformClient.GetTargets(context.Background(), targetType)
	if err != nil {
		return nil, err
	}
	return GetTargetsResponse.Targets, nil
}

func (nspc *NetworkServicePlateformClient) GetTargetsWithType(t nspAPI.Target_Type) ([]*nspAPI.Target, error) {
	targetType := &nspAPI.TargetType{
		Type: t,
	}
	GetTargetsResponse, err := nspc.networkServicePlateformClient.GetTargets(context.Background(), targetType)
	if err != nil {
		return nil, err
	}
	return GetTargetsResponse.Targets, nil
}

func (nspc *NetworkServicePlateformClient) connect(ipamServiceIPPort string) error {
	var err error
	nspc.conn, err = grpc.Dial(ipamServiceIPPort, grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil
	}

	nspc.networkServicePlateformClient = nspAPI.NewNetworkServicePlateformServiceClient(nspc.conn)
	return nil
}

func (nspc *NetworkServicePlateformClient) Delete() error {
	if nspc.conn == nil {
		return nil
	}
	return nspc.conn.Close()
}

func NewNetworkServicePlateformClient(serviceIPPort string) (*NetworkServicePlateformClient, error) {
	networkServicePlateformClient := &NetworkServicePlateformClient{}
	err := networkServicePlateformClient.connect(serviceIPPort)
	if err != nil {
		return nil, err
	}
	return networkServicePlateformClient, nil
}
