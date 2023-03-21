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

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/nsp/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	nspAPI.UnimplementedTargetRegistryServer
	TargetRegistry types.TargetRegistry
}

// NewServer -
func NewServer(targetRegistry types.TargetRegistry) nspAPI.TargetRegistryServer {
	networkServicePlateformService := &Server{
		TargetRegistry: targetRegistry,
	}

	return networkServicePlateformService
}

func (s *Server) Register(ctx context.Context, target *nspAPI.Target) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.TargetRegistry.Set(ctx, target)
}

func (s *Server) Unregister(ctx context.Context, target *nspAPI.Target) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, s.TargetRegistry.Remove(ctx, target)
}

func (s *Server) Watch(t *nspAPI.Target, watcher nspAPI.TargetRegistry_WatchServer) error {
	targetWatcher, err := s.TargetRegistry.Watch(context.TODO(), t)
	if err != nil {
		return err
	}
	s.watcher(watcher, targetWatcher.ResultChan())
	targetWatcher.Stop()
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
				log.Logger.Error(err, "Sending TrenchResponse")
			}
		case <-watcher.Context().Done():
			return
		}
	}
}
