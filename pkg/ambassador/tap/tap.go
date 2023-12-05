/*
Copyright (c) 2021-2023 Nordix Foundation

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

package tap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream/registry"
	"github.com/nordix/meridio/pkg/ambassador/tap/trench"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Tap implements ambassadorAPI.TapServer
type Tap struct {
	ambassadorAPI.UnimplementedTapServer
	TargetName              string
	Namespace               string
	NodeName                string
	NetworkServiceClient    networkservice.NetworkServiceClient
	MonitorConnectionClient networkservice.MonitorConnectionClient
	NSPServiceName          string
	NSPServicePort          int
	NSPEntryTimeout         time.Duration
	GRPCMaxBackoff          time.Duration
	NetUtils                networking.Utils
	StreamRegistry          types.Registry
	currentTrench           types.Trench
	mu                      sync.Mutex
	logger                  logr.Logger
}

func New(targetName string,
	namespace string,
	nodeName string,
	networkServiceClient networkservice.NetworkServiceClient,
	monitorConnectionClient networkservice.MonitorConnectionClient,
	nspServiceName string,
	nspServicePort int,
	nspEntryTimeout time.Duration,
	grpcMaxBackoff time.Duration,
	netUtils networking.Utils) (*Tap, error) {
	tap := &Tap{
		TargetName:              targetName,
		NetworkServiceClient:    networkServiceClient,
		MonitorConnectionClient: monitorConnectionClient,
		Namespace:               namespace,
		NodeName:                nodeName,
		NSPServiceName:          nspServiceName,
		NSPServicePort:          nspServicePort,
		NSPEntryTimeout:         nspEntryTimeout,
		GRPCMaxBackoff:          grpcMaxBackoff,
		NetUtils:                netUtils,
		logger:                  log.Logger.WithValues("class", "Tap"),
	}
	tap.StreamRegistry = registry.New()
	return tap, nil
}

func (tap *Tap) Open(ctx context.Context, s *ambassadorAPI.Stream) (*emptypb.Empty, error) {
	tap.mu.Lock()
	defer tap.mu.Unlock()
	err := checkStream(s)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	tap.logger.Info("Open", "stream", s)
	// set the trench if tap.currentTrench in nil, get if s.conduit.trench == currentTrench
	// return an error if s.conduit.trench != currentTrench
	trench, err := tap.setTrench(s.GetConduit().GetTrench())
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("open failed to set trench: %w", err)
	}
	// add/get conduit (get if already existing)
	// will be connected when the trench will be ready
	conduit, err := trench.AddConduit(context.TODO(), s.GetConduit())
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("open failed to add conduit: %w", err)
	}
	// add/get a stream (get if already existing)
	// will be opened when the conduit will be ready
	err = conduit.AddStream(context.TODO(), s)
	if err != nil {
		return &emptypb.Empty{}, fmt.Errorf("open failed to add stream: %w", err)
	}
	return &emptypb.Empty{}, nil
}

func (tap *Tap) Close(ctx context.Context, s *ambassadorAPI.Stream) (*emptypb.Empty, error) {
	err := checkStream(s)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	tap.logger.Info("Close", "stream", s)
	go func() {
		tap.mu.Lock()
		defer tap.mu.Unlock()
		if tap.currentTrench == nil || !tap.currentTrench.Equals(s.GetConduit().GetTrench()) {
			return
		}
		// todo: set timeout (the env variable) instead of 10 seconds
		context, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()
		conduit := tap.currentTrench.GetConduit(s.GetConduit())
		if conduit == nil {
			return
		}
		err := conduit.RemoveStream(context, s) // todo: retry
		if err != nil {
			tap.logger.Error(err, "removing stream")
		}
		if len(conduit.GetStreams()) > 0 {
			return
		}
		// remove the conduit if it doesn't contain any more stream
		err = tap.currentTrench.RemoveConduit(context, s.GetConduit()) // todo: retry
		if err != nil {
			tap.logger.Error(err, "removing conduit")
		}
		if len(tap.currentTrench.GetConduits()) > 0 {
			return
		}
		// delete the trench if it doesn't contain any more conduit
		err = tap.currentTrench.Delete(context)
		if err != nil {
			tap.logger.Error(err, "deleting trench")
		}
		tap.currentTrench = nil
	}()

	return &emptypb.Empty{}, nil
}

func (tap *Tap) Watch(filter *ambassadorAPI.Stream, watcher ambassadorAPI.Tap_WatchServer) error {
	logger := tap.logger.WithValues("filter", filter, "watcher", watcher)
	logger.V(1).Info("Watch")
	targetWatcher, err := tap.StreamRegistry.Watch(context.TODO(), filter)
	if err != nil {
		return fmt.Errorf("failed to create stream registry watch: %w", err)
	}
	tap.watcher(watcher, targetWatcher.ResultChan())
	logger.V(1).Info("Watch stopped")
	targetWatcher.Stop()
	return nil
}

func (tap *Tap) setTrench(t *ambassadorAPI.Trench) (types.Trench, error) {
	if tap.currentTrench != nil {
		if tap.currentTrench.Equals(t) {
			return tap.currentTrench, nil
		}
		return nil, fmt.Errorf("another trench is already connected")
	}
	res, err := trench.New(t,
		tap.TargetName,
		tap.Namespace,
		tap.NodeName,
		tap.NetworkServiceClient,
		tap.MonitorConnectionClient,
		tap.StreamRegistry,
		tap.NSPServiceName,
		tap.NSPServicePort,
		tap.NSPEntryTimeout,
		tap.GRPCMaxBackoff,
		tap.NetUtils)
	if err != nil {
		return nil, fmt.Errorf("failed to create trench: %w", err)
	}
	tap.currentTrench = res
	return res, nil
}

func (tap *Tap) Delete(ctx context.Context) error {
	if tap.currentTrench == nil {
		return nil
	}
	tap.logger.Info("Delete", "currentTrench", tap.currentTrench)
	if err := tap.currentTrench.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete trench: %w", err)
	}
	return nil
}

func (tap *Tap) watcher(watcher ambassadorAPI.Tap_WatchServer, ch <-chan []*ambassadorAPI.StreamStatus) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&ambassadorAPI.StreamResponse{
				StreamStatus: event,
			})
			if err != nil {
				tap.logger.Error(err, "sending TrenchResponse")
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

// check if name are not empty and if conduit and trench are not nil
func checkStream(s *ambassadorAPI.Stream) error {
	if s == nil {
		return fmt.Errorf("stream cannot be nil")
	}
	if s.GetConduit() == nil {
		return fmt.Errorf("conduit cannot be nil")
	}
	if s.GetConduit().Trench == nil {
		return fmt.Errorf("trench cannot be nil")
	}
	return nil
}
