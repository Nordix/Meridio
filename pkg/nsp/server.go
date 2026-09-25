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
	err := s.TargetRegistry.Set(ctx, target)
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed to set target (%s): %w", target.String(), err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) Unregister(ctx context.Context, target *nspAPI.Target) (*emptypb.Empty, error) {
	err := s.TargetRegistry.Remove(ctx, target)
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed to remove target (%s): %w", target.String(), err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) Watch(t *nspAPI.Target, watcher nspAPI.TargetRegistry_WatchServer) error {
	targetWatcher, err := s.TargetRegistry.Watch(context.TODO(), t)
	// The registry may return a non-nil watcher even alongside an error (e.g.
	// the sqlite registry registers the watcher before its initial read can
	// fail). Ensure Stop() runs on every exit path so such a watcher is not
	// left registered (which would keep receiving fan-out notifications with no
	// reader). The nil-check is required because other registry layers (e.g.
	// keepalive with no backing registry) might return a nil watcher on error.
	if targetWatcher != nil {
		defer targetWatcher.Stop()
	}
	if err != nil {
		return fmt.Errorf("failed getting target watcher (%s): %w", t.String(), err)
	}
	s.watcher(watcher, targetWatcher.ResultChan())
	return nil
}

func (s *Server) watcher(watcher nspAPI.TargetRegistry_WatchServer, ch <-chan []*nspAPI.Target) {
	for {
		select {
		case event := <-ch:
			// Per the grpc.ServerStream.SendMsg contract ("On error, SendMsg
			// aborts the stream and the error is returned directly"), a non-nil
			// Send error means this watch stream is aborted and unusable. Return
			// to tear the watcher down promptly (the deferred Stop() in Watch()
			// deregisters it) instead of looping and re-sending on a dead stream.
			// The client wraps its Watch in a retry loop and will reconnect,
			// obtaining a fresh full target set from the registry's initial read.
			//
			// TODO: Send has no deadline. A slow or half-open consumer can block
			// this goroutine (and thus the drain of ch) until the stream context
			// is cancelled, which for a half-open connection may take until the
			// transport keepalive/timeout fires. Consider bounding Send with a
			// timeout and treating a timeout as stream death (return) as well.
			if err := watcher.Send(&nspAPI.TargetResponse{
				Targets: event,
			}); err != nil {
				log.Logger.Error(err, "Sending TargetResponse, closing watch stream")
				return
			}
		case <-watcher.Context().Done():
			return
		}
	}
}
