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
	targetWatcher, err := s.TargetRegistry.Watch(watcher.Context(), t)
	// Stop() runs on every exit path. The registry returns no watcher on error
	// today, so the nil check and stopping on the error path are defensive:
	// they keep a registry that returns both from leaving a registered watcher
	// that receives notifications with no reader.
	if targetWatcher != nil {
		defer targetWatcher.Stop()
	}
	if err != nil {
		return fmt.Errorf("failed getting target watcher (%s): %w", t.String(), err)
	}
	return s.watcher(watcher, targetWatcher.ResultChan())
}

// watcher sends every target list from ch to the client. It returns nil when
// the client goes away, and the error when a send fails, so the stream ends
// with that error's status instead of OK.
func (s *Server) watcher(watcher nspAPI.TargetRegistry_WatchServer, ch <-chan []*nspAPI.Target) error {
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
				if watcher.Context().Err() != nil {
					return nil // client went away during the send: normal end of the stream
				}
				// gRPC does not log errors returned by handlers, so log it here.
				log.Logger.Error(err, "Sending TargetResponse, closing watch stream")
				return fmt.Errorf("failed sending target response: %w", err)
			}
		case <-watcher.Context().Done():
			return nil // client went away: normal end of the stream
		}
	}
}
