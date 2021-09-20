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
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Server struct {
	Listener       net.Listener
	Server         *grpc.Server
	Port           int
	targets        *targetList
	monitorStreams sync.Map // map[nspAPI.Target_Type]map[nspAPI.NetworkServicePlateformService_MonitorServer]bool
	// monitorStreams sync.Map // map[nspAPI.NetworkServicePlateformService_MonitorServer]bool
}

// NewServer -
func NewServer(port int) (*Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		logrus.Errorf("NSP Service: failed to listen: %v", err)
		return nil, err
	}

	s := grpc.NewServer()

	networkServicePlateformService := &Server{
		Listener: lis,
		Server:   s,
		Port:     port,
		targets: &targetList{
			targets: map[nspAPI.Target_Type][]*nspAPI.Target{},
		},
	}

	nspAPI.RegisterNetworkServicePlateformServiceServer(s, networkServicePlateformService)

	return networkServicePlateformService, nil
}

// Start -
func (s *Server) Start() {
	logrus.Infof("NSP Service: Start the service (port: %v)", s.Port)
	if err := s.Server.Serve(s.Listener); err != nil {
		logrus.Errorf("NSP Service: failed to serve: %v", err)
	}
}

func (s *Server) Register(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := s.addTarget(target)
	return &empty.Empty{}, err
}

func (s *Server) Unregister(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := s.removeTarget(target)
	return &empty.Empty{}, err
}

func (s *Server) Update(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	err := s.updateTarget(target)
	return &empty.Empty{}, err
}

// todo
func (s *Server) Monitor(tt *nspAPI.TargetType, stream nspAPI.NetworkServicePlateformService_MonitorServer) error {
	targetType := tt.GetType()
	streams, err := s.getStreams(targetType)
	if err != nil {
		logrus.Infof("Monitor: err: %v", err)
		return err
	}

	logrus.Debugf("Monitor: targetType: %v, stream: %v", targetType, stream)
	streams.Store(stream, true)
	for _, target := range s.targets.Get(targetType) {
		s.notifyMonitorStream(stream, target, nspAPI.TargetEvent_Register)
	}
	<-stream.Context().Done()
	streams.Delete(stream)
	return nil
}

func (s *Server) GetTargets(ctx context.Context, tt *nspAPI.TargetType) (*nspAPI.GetTargetsResponse, error) {
	response := &nspAPI.GetTargetsResponse{
		Targets: s.targets.Get(tt.GetType()),
	}
	return response, nil
}

func (s *Server) streamAlive(stream nspAPI.NetworkServicePlateformService_MonitorServer, streams *sync.Map) bool {
	value, ok := streams.Load(stream)
	return ok && value.(bool)
}

func (s *Server) addTarget(target *nspAPI.Target) error {
	logrus.Infof("Add Target: %v", target)
	err := s.targets.Add(target)
	if err != nil {
		return err
	}
	s.notifyMonitorStreams(target, nspAPI.TargetEvent_Register)
	return nil
}

func (s *Server) removeTarget(target *nspAPI.Target) error {
	t, err := s.targets.Remove(target)
	if err != nil {
		return err
	}
	logrus.Infof("Remove Target: %v", target)
	s.notifyMonitorStreams(t, nspAPI.TargetEvent_Unregister)
	return nil
}

func (s *Server) updateTarget(target *nspAPI.Target) error {
	err := s.targets.Update(target)
	if err != nil {
		return err
	}
	logrus.Infof("Update Target: %v", target)
	s.notifyMonitorStreams(target, nspAPI.TargetEvent_Updated)
	return nil
}

func (s *Server) getStreams(targetType nspAPI.Target_Type) (*sync.Map, error) {
	value, _ := s.monitorStreams.LoadOrStore(targetType, &sync.Map{})
	return value.(*sync.Map), nil
}

func (s *Server) notifyMonitorStreams(target *nspAPI.Target, eventStatus nspAPI.TargetEvent_Status) {
	logrus.Debugf("notifyMonitorStreams: target: %v,", target)
	streams, err := s.getStreams(target.GetType())
	if err != nil {
		logrus.Infof("notifyMonitorStreams: err: %v", err)
		return
	}
	streams.Range(func(key interface{}, value interface{}) bool {
		s.notifyMonitorStream(key.(nspAPI.NetworkServicePlateformService_MonitorServer), target, eventStatus)
		return true
	})
}

func (s *Server) notifyMonitorStream(stream nspAPI.NetworkServicePlateformService_MonitorServer, target *nspAPI.Target, eventStatus nspAPI.TargetEvent_Status) {
	streams, err := s.getStreams(target.GetType())
	if err != nil {
		logrus.Infof("notifyMonitorStream: err: %v", err)
		return
	}
	if !s.streamAlive(stream, streams) {
		return
	}
	targetEvent := &nspAPI.TargetEvent{
		Target: target,
		Status: eventStatus,
	}
	err = stream.Send(targetEvent)
	if err != nil {
		logrus.Infof("notifyMonitorStream: send err: %v", err)
		s.monitorStreams.Store(stream, false)
	}
}
