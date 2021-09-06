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
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type NetworkServicePlateformService struct {
	Listener       net.Listener
	Server         *grpc.Server
	Port           int
	targets        *targetList
	monitorStreams sync.Map
}

func (nsps *NetworkServicePlateformService) addTarget(target *nspAPI.Target) error {
	logrus.Infof("Add Target: %v", target)
	err := nsps.targets.Add(target)
	if err != nil {
		return err
	}
	target.Status = nspAPI.Status_Register
	nsps.notifyMonitorStreams(target)
	return nil
}

func (nsps *NetworkServicePlateformService) removeTarget(target *nspAPI.Target) error {
	t, err := nsps.targets.Remove(target)
	if err != nil {
		return err
	}
	logrus.Infof("Remove Target: %v", target)
	t.Status = nspAPI.Status_Unregister
	nsps.notifyMonitorStreams(t)
	return nil
}

func (nsps *NetworkServicePlateformService) notifyMonitorStreams(target *nspAPI.Target) {
	logrus.Debugf("notifyMonitorStreams: target: %v,", target)
	streams, err := nsps.getStreams(target.GetContext()[TARGETTYPE])
	if err != nil {
		logrus.Infof("notifyMonitorStreams: err: %v", err)
		return
	}
	streams.Range(func(key interface{}, value interface{}) bool {
		nsps.notifyMonitorStream(key.(nspAPI.NetworkServicePlateformService_MonitorServer), target)
		return true
	})
}

func (nsps *NetworkServicePlateformService) notifyMonitorStream(stream nspAPI.NetworkServicePlateformService_MonitorServer, target *nspAPI.Target) {
	streams, err := nsps.getStreams(target.GetContext()[TARGETTYPE])
	if err != nil {
		logrus.Infof("notifyMonitorStream: err: %v", err)
		return
	}
	if !nsps.streamAlive(stream, streams) {
		return
	}
	logrus.Debugf("notifyMonitorStream: target: %v, stream: %v", target, stream)
	err = stream.Send(target)
	if err != nil {
		logrus.Infof("notifyMonitorStream: send err: %v", err)
		streams.Store(stream, false)
	}
}

func (nsps *NetworkServicePlateformService) Register(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := nsps.addTarget(target)
	return &empty.Empty{}, err
}

func (nsps *NetworkServicePlateformService) Unregister(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := nsps.removeTarget(target)
	return &empty.Empty{}, err
}

func (nsps *NetworkServicePlateformService) Monitor(tt *nspAPI.TargetType, stream nspAPI.NetworkServicePlateformService_MonitorServer) error {
	targetType := tt.GetType()
	streams, err := nsps.getStreams(targetType)
	if err != nil {
		logrus.Infof("Monitor: err: %v", err)
		return err
	}

	logrus.Debugf("Monitor: targetType: \"%v\", stream: %v", targetType, stream)
	streams.Store(stream, true)
	for _, target := range nsps.targets.Get(targetType) {
		nsps.notifyMonitorStream(stream, target)
	}
	<-stream.Context().Done()
	streams.Delete(stream)
	return nil
}

func (nsps *NetworkServicePlateformService) getStreams(targetType string) (*sync.Map, error) {
	if targetType != "" {
		value, _ := nsps.monitorStreams.LoadOrStore(targetType, &sync.Map{})
		return value.(*sync.Map), nil
	}
	return nil, errors.New(`"` + targetType + `" monitor streams not found`)
}

func (nsps *NetworkServicePlateformService) streamAlive(stream nspAPI.NetworkServicePlateformService_MonitorServer, streams *sync.Map) bool {
	value, ok := streams.Load(stream)
	return ok && value.(bool)
}

func (nsps *NetworkServicePlateformService) GetTargets(ctx context.Context, tt *nspAPI.TargetType) (*nspAPI.GetTargetsResponse, error) {
	response := &nspAPI.GetTargetsResponse{
		Targets: nsps.targets.Get(tt.GetType()),
	}
	return response, nil
}

// Start -
func (nsps *NetworkServicePlateformService) Start() {
	logrus.Infof("NSP Service: Start the service (port: %v)", nsps.Port)
	if err := nsps.Server.Serve(nsps.Listener); err != nil {
		logrus.Errorf("NSP Service: failed to serve: %v", err)
	}
}

// NewNetworkServicePlateformService -
func NewNetworkServicePlateformService(port int) (*NetworkServicePlateformService, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		logrus.Errorf("NSP Service: failed to listen: %v", err)
		return nil, err
	}

	s := grpc.NewServer()

	networkServicePlateformService := &NetworkServicePlateformService{
		Listener: lis,
		Server:   s,
		Port:     port,
		targets: &targetList{
			targets: map[string][]*target{},
		},
	}

	nspAPI.RegisterNetworkServicePlateformServiceServer(s, networkServicePlateformService)

	return networkServicePlateformService, nil
}
