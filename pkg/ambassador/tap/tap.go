/*
Copyright (c) 2021-2022 Nordix Foundation

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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	tapAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream/registry"
	"github.com/nordix/meridio/pkg/ambassador/tap/trench"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

type Tap struct {
	TargetName           string
	Namespace            string
	NodeName             string
	NetworkServiceClient networkservice.NetworkServiceClient
	NSPServiceName       string
	NSPServicePort       int
	NetUtils             networking.Utils
	StreamRegistry       types.Registry
	currentTrench        types.Trench
	mu                   sync.Mutex
}

func New(targetName string,
	namespace string,
	nodeName string,
	networkServiceClient networkservice.NetworkServiceClient,
	nspServiceName string,
	nspServicePort int,
	netUtils networking.Utils) (*Tap, error) {
	tap := &Tap{
		NetworkServiceClient: networkServiceClient,
		Namespace:            namespace,
		NodeName:             nodeName,
		NSPServiceName:       nspServiceName,
		NSPServicePort:       nspServicePort,
		NetUtils:             netUtils,
	}
	tap.StreamRegistry = registry.New()
	return tap, nil
}

func (tap *Tap) Open(ctx context.Context, s *nspAPI.Stream) (*empty.Empty, error) {
	tap.mu.Lock()
	defer tap.mu.Unlock()
	err := checkStream(s)
	if err != nil {
		return &empty.Empty{}, err
	}
	trench, err := tap.setTrench(s.GetConduit().GetTrench())
	if err != nil {
		return &empty.Empty{}, err
	}
	conduit, err := trench.AddConduit(context.TODO(), s.GetConduit())
	if err != nil {
		return &empty.Empty{}, err
	}
	_, err = conduit.AddStream(context.TODO(), s)
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, err
}

func (tap *Tap) Close(ctx context.Context, s *nspAPI.Stream) (*empty.Empty, error) {
	err := checkStream(s)
	if err != nil {
		return &empty.Empty{}, err
	}
	go func() {
		tap.mu.Lock()
		defer tap.mu.Unlock()
		if tap.currentTrench == nil || !tap.currentTrench.Equals(s.GetConduit().GetTrench()) {
			return
		}
		context, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()
		conduit := tap.currentTrench.GetConduit(s.GetConduit())
		if conduit == nil {
			return
		}
		err := conduit.RemoveStream(context, s)
		if err != nil {
			logrus.Errorf("Error removing stream: %v", err)
		}
		if len(conduit.GetStreams()) > 0 {
			return
		}
		err = tap.currentTrench.RemoveConduit(context, s.GetConduit())
		if err != nil {
			logrus.Errorf("Error removing conduit: %v", err)
		}
		if len(tap.currentTrench.GetConduits()) > 0 {
			return
		}
		err = tap.currentTrench.Delete(context)
		if err != nil {
			logrus.Errorf("Error deleting trench: %v", err)
		}
		tap.currentTrench = nil
	}()

	return &empty.Empty{}, nil
}

func (tap *Tap) Watch(filter *nspAPI.Stream, watcher tapAPI.Tap_WatchServer) error {
	targetWatcher, err := tap.StreamRegistry.Watch(context.TODO(), filter)
	if err != nil {
		return err
	}
	tap.watcher(watcher, targetWatcher.ResultChan())
	targetWatcher.Stop()
	return nil
}

func (tap *Tap) setTrench(t *nspAPI.Trench) (types.Trench, error) {
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
		tap.StreamRegistry,
		tap.NSPServiceName,
		tap.NSPServicePort,
		tap.NetUtils)
	if err != nil {
		return nil, err
	}
	tap.currentTrench = res
	return res, nil
}

func (tap *Tap) Delete(ctx context.Context) error {
	if tap.currentTrench == nil {
		return nil
	}
	return tap.currentTrench.Delete(ctx)
}

func (tap *Tap) watcher(watcher tapAPI.Tap_WatchServer, ch <-chan []*tapAPI.StreamStatus) {
	for {
		select {
		case event := <-ch:
			err := watcher.Send(&tapAPI.StreamResponse{
				StreamStatus: event,
			})
			if err != nil {
				logrus.Errorf("err sending TrenchResponse: %v", err)
			}
		case <-watcher.Context().Done():
			return
		}
	}
}

func checkStream(s *nspAPI.Stream) error {
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
