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

package manager

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/sirupsen/logrus"
)

const (
	channelBufferSize = 10
)

// Server implements the ConfigurationManagerServer nsp service
type Server struct {
	WatcherRegistry WatcherRegistry
}

// NewServer is the constructor of Server
func NewServer(watcherRegistry WatcherRegistry) nspAPI.ConfigurationManagerServer {
	networkServicePlateformService := &Server{
		WatcherRegistry: watcherRegistry,
	}

	return networkServicePlateformService
}

func (s *Server) WatchTrench(trench *nspAPI.Trench, watcher nspAPI.ConfigurationManager_WatchTrenchServer) error {
	ch := make(chan *nspAPI.Trench, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(trench, ch)
	if err != nil {
		return err
	}
	s.trenchWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) WatchConduit(conduit *nspAPI.Conduit, watcher nspAPI.ConfigurationManager_WatchConduitServer) error {
	ch := make(chan []*nspAPI.Conduit, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(conduit, ch)
	if err != nil {
		return err
	}
	s.conduitWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) WatchStream(stream *nspAPI.Stream, watcher nspAPI.ConfigurationManager_WatchStreamServer) error {
	ch := make(chan []*nspAPI.Stream, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(stream, ch)
	if err != nil {
		return err
	}
	s.streamWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) WatchFlow(flow *nspAPI.Flow, watcher nspAPI.ConfigurationManager_WatchFlowServer) error {
	ch := make(chan []*nspAPI.Flow, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(flow, ch)
	if err != nil {
		return err
	}
	s.flowWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) WatchVip(vip *nspAPI.Vip, watcher nspAPI.ConfigurationManager_WatchVipServer) error {
	ch := make(chan []*nspAPI.Vip, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(vip, ch)
	if err != nil {
		return err
	}
	s.vipWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) WatchAttractor(attractor *nspAPI.Attractor, watcher nspAPI.ConfigurationManager_WatchAttractorServer) error {
	ch := make(chan []*nspAPI.Attractor, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(attractor, ch)
	if err != nil {
		return err
	}
	s.attractorWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) WatchGateway(gateway *nspAPI.Gateway, watcher nspAPI.ConfigurationManager_WatchGatewayServer) error {
	ch := make(chan []*nspAPI.Gateway, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(gateway, ch)
	if err != nil {
		return err
	}
	s.gatewayWatcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) trenchWatcher(watcher nspAPI.ConfigurationManager_WatchTrenchServer, ch <-chan *nspAPI.Trench) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.TrenchResponse{
				Trench: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func (s *Server) conduitWatcher(watcher nspAPI.ConfigurationManager_WatchConduitServer, ch <-chan []*nspAPI.Conduit) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.ConduitResponse{
				Conduits: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func (s *Server) streamWatcher(watcher nspAPI.ConfigurationManager_WatchStreamServer, ch <-chan []*nspAPI.Stream) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.StreamResponse{
				Streams: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func (s *Server) flowWatcher(watcher nspAPI.ConfigurationManager_WatchFlowServer, ch <-chan []*nspAPI.Flow) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.FlowResponse{
				Flows: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func (s *Server) vipWatcher(watcher nspAPI.ConfigurationManager_WatchVipServer, ch <-chan []*nspAPI.Vip) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.VipResponse{
				Vips: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func (s *Server) attractorWatcher(watcher nspAPI.ConfigurationManager_WatchAttractorServer, ch <-chan []*nspAPI.Attractor) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.AttractorResponse{
				Attractors: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func (s *Server) gatewayWatcher(watcher nspAPI.ConfigurationManager_WatchGatewayServer, ch <-chan []*nspAPI.Gateway) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.GatewayResponse{
				Gateways: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}
