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

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/types"
	"github.com/sirupsen/logrus"
)

const (
	channelBufferSize = 10
)

type WatcherRegistry interface {
	RegisterWatcher(toWatch *nspAPI.Target, ch chan<- []*nspAPI.Target) error
	UnregisterWatcher(ch chan<- []*nspAPI.Target)
}

type Server struct {
	TargetRegistry  types.TargetRegistry
	WatcherRegistry WatcherRegistry
}

// NewServer -
func NewServer(targetRegistry types.TargetRegistry, watcherRegistry WatcherRegistry) nspAPI.TargetRegistryServer {
	networkServicePlateformService := &Server{
		TargetRegistry:  targetRegistry,
		WatcherRegistry: watcherRegistry,
	}

	return networkServicePlateformService
}

func (s *Server) Register(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	logrus.Infof("Register: %v", target)
	s.TargetRegistry.Set(target)
	return &empty.Empty{}, nil
}

func (s *Server) Unregister(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	logrus.Infof("Unregister: %v", target)
	s.TargetRegistry.Remove(target)
	return &empty.Empty{}, nil
}

func (s *Server) Update(ctx context.Context, target *nspAPI.Target) (*empty.Empty, error) {
	logrus.Infof("Update: %v", target)
	s.TargetRegistry.Set(target)
	return &empty.Empty{}, nil
}

func (s *Server) Watch(t *nspAPI.Target, watcher nspAPI.TargetRegistry_WatchServer) error {
	ch := make(chan []*nspAPI.Target, channelBufferSize)
	err := s.WatcherRegistry.RegisterWatcher(t, ch)
	if err != nil {
		return err
	}
	s.watcher(watcher, ch)
	s.WatcherRegistry.UnregisterWatcher(ch)
	return nil
}

func (s *Server) watcher(watcher nspAPI.TargetRegistry_WatchServer, ch <-chan []*nspAPI.Target) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&nspAPI.TargetResponse{
				Targets: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}
