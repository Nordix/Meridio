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

package target

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	targetAPI "github.com/nordix/meridio/api/target"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/target/conduit"
	"github.com/nordix/meridio/pkg/target/stream"
	"github.com/nordix/meridio/pkg/target/trench"
	"github.com/nordix/meridio/pkg/target/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Ambassador struct {
	context                  context.Context
	listener                 net.Listener
	server                   *grpc.Server
	port                     int
	vips                     []string
	trenches                 []types.Trench
	trenchNamespace          string
	config                   *Config
	watchConduitsSubscribers sync.Map // map[nspAPI.Ambassador_WatchConduitsServer]struct{}
	watchStreamsSubscribers  sync.Map // map[nspAPI.Ambassador_WatchStreamsServer]struct{}
	conduitWatcher           chan *targetAPI.ConduitEvent
	streamWatcher            chan *targetAPI.StreamEvent
	mu                       sync.Mutex
}

func NewAmbassador(port int, trenchNamespace string, config *Config) (*Ambassador, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()

	ambassador := &Ambassador{
		listener:        lis,
		server:          s,
		port:            port,
		vips:            []string{},
		trenches:        []types.Trench{},
		trenchNamespace: trenchNamespace,
		config:          config,
		conduitWatcher:  make(chan *targetAPI.ConduitEvent, 10),
		streamWatcher:   make(chan *targetAPI.StreamEvent, 10),
	}

	targetAPI.RegisterAmbassadorServer(s, ambassador)

	return ambassador, nil
}

func (a *Ambassador) Connect(ctx context.Context, c *targetAPI.Conduit) (*empty.Empty, error) {
	logrus.Infof("Connect to conduit: %v ; trench %v (%v)", c.GetNetworkServiceName(), c.GetTrench().GetName(), a.trenchNamespace)
	var err error
	t := a.getTrench(c.GetTrench().GetName(), a.trenchNamespace)
	if t == nil {
		if len(a.trenches) >= 1 {
			return &empty.Empty{}, errors.New("a trench is already connected")
		}
		t, err = trench.New(c.GetTrench().GetName(), a.trenchNamespace, a.config.configMapName, a.config.nspServiceName, a.config.nspServicePort)
		if err != nil {
			return &empty.Empty{}, err
		}
		err = a.addTrench(t)
		if err != nil {
			errDelete := t.Delete(ctx)
			if err != nil {
				return &empty.Empty{}, fmt.Errorf("%w; %v", err, errDelete)
			}
			return &empty.Empty{}, err
		}
	}
	_, err = conduit.New(ctx, c.GetNetworkServiceName(), t, a.config.apiClient, a.config.nsmConfig, a.conduitWatcher, a.config.netUtils) // todo: api
	return &empty.Empty{}, err
}

func (a *Ambassador) Disconnect(ctx context.Context, c *targetAPI.Conduit) (*empty.Empty, error) {
	logrus.Infof("Disconnect from conduit: %v ; trench %v (%v)", c.GetNetworkServiceName(), c.GetTrench().GetName(), a.trenchNamespace)
	trench := a.getTrench(c.GetTrench().GetName(), a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, errors.New("not connected to the trench")
	}
	conduit := trench.GetConduit(c.GetNetworkServiceName())
	if c == nil {
		return &empty.Empty{}, errors.New("not connected to the conduit")
	}
	err := trench.RemoveConduit(ctx, conduit)
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, a.deleteTrench(trench) // todo
}

func (a *Ambassador) Request(ctx context.Context, s *targetAPI.Stream) (*empty.Empty, error) {
	logrus.Infof("Request stream: %v ; conduit: %v ; trench %v (%v)", s.GetName(), s.GetConduit().GetNetworkServiceName(), s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	trench := a.getTrench(s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, errors.New("not connected to the trench")
	}
	conduit := trench.GetConduit(s.Conduit.NetworkServiceName)
	if conduit == nil {
		return &empty.Empty{}, errors.New("not connected to the conduit")
	}
	_, err := stream.New(ctx, s.Name, conduit, a.streamWatcher) // todo: api
	return &empty.Empty{}, err
}

func (a *Ambassador) Close(ctx context.Context, s *targetAPI.Stream) (*empty.Empty, error) {
	logrus.Infof("Close stream: %v ; conduit: %v ; trench %v (%v)", s.GetName(), s.GetConduit().GetNetworkServiceName(), s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	trench := a.getTrench(s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	conduit := trench.GetConduit(s.Conduit.NetworkServiceName)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	stream := conduit.GetStream(s.Name)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	return &empty.Empty{}, conduit.RemoveStream(ctx, stream)
}

func (a *Ambassador) WatchConduits(empty *empty.Empty, stream targetAPI.Ambassador_WatchConduitsServer) error {
	a.watchConduitsSubscribers.Store(stream, struct{}{})
	for _, conduit := range a.trenches[0].GetConduits() {
		a.notifyConduitsSubscriber(stream, &targetAPI.ConduitEvent{
			Conduit: &targetAPI.Conduit{
				NetworkServiceName: conduit.GetName(),
				Trench: &targetAPI.Trench{
					Name: conduit.GetTrench().GetName(),
				},
			},
			ConduitEventStatus: targetAPI.ConduitEventStatus_Connect,
		})
	}
	<-stream.Context().Done()
	a.watchConduitsSubscribers.Delete(stream)
	return nil
}

func (a *Ambassador) WatchStreams(empty *empty.Empty, stream targetAPI.Ambassador_WatchStreamsServer) error {
	a.watchStreamsSubscribers.Store(stream, struct{}{})
	for _, conduit := range a.trenches[0].GetConduits() {
		for _, st := range conduit.GetStreams() {
			a.notifyStreamsSubscriber(stream, &targetAPI.StreamEvent{
				Stream: &targetAPI.Stream{
					Conduit: &targetAPI.Conduit{
						NetworkServiceName: st.GetConduit().GetName(),
						Trench: &targetAPI.Trench{
							Name: st.GetConduit().GetTrench().GetName(),
						},
					},
				},
				StreamEventStatus: targetAPI.StreamEventStatus_Request,
			})
		}
	}
	<-stream.Context().Done()
	a.watchStreamsSubscribers.Delete(stream)
	return nil
}

func (a *Ambassador) Start(ctx context.Context) error {
	a.context = ctx
	a.config.apiClient = nsm.NewAPIClient(a.context, a.config.nsmConfig)
	go a.watcher()
	return a.server.Serve(a.listener)
}

func (a *Ambassador) Delete() error {
	a.context.Done()
	a.server.Stop()
	for _, trench := range a.trenches {
		err := trench.Delete(context.Background())
		if err != nil {
			logrus.Errorf("Error deleting a trench: %v", err)
		}
	}
	return nil
}

func (a *Ambassador) watcher() {
	for {
		select {
		case conduitEvent := <-a.conduitWatcher:
			a.notifyConduitsSubscribers(conduitEvent)
		case streamEvent := <-a.streamWatcher:
			a.notifyStreamsSubscribers(streamEvent)
		case <-a.context.Done():
			return
		}
	}
}

func (a *Ambassador) notifyConduitsSubscribers(conduitEvent *targetAPI.ConduitEvent) {
	a.watchConduitsSubscribers.Range(func(key interface{}, value interface{}) bool {
		a.notifyConduitsSubscriber(key.(targetAPI.Ambassador_WatchConduitsServer), conduitEvent)
		return true
	})
}

func (a *Ambassador) notifyConduitsSubscriber(subscriber targetAPI.Ambassador_WatchConduitsServer, conduitEvent *targetAPI.ConduitEvent) {
	_ = subscriber.Send(conduitEvent)
}

func (a *Ambassador) notifyStreamsSubscribers(streamEvent *targetAPI.StreamEvent) {
	a.watchStreamsSubscribers.Range(func(key interface{}, value interface{}) bool {
		a.notifyStreamsSubscriber(key.(targetAPI.Ambassador_WatchStreamsServer), streamEvent)
		return true
	})
}

func (a *Ambassador) notifyStreamsSubscriber(subscriber targetAPI.Ambassador_WatchStreamsServer, streamEvent *targetAPI.StreamEvent) {
	_ = subscriber.Send(streamEvent)
}

func (a *Ambassador) addTrench(trench types.Trench) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	index := a.getIndex(trench.GetName(), trench.GetNamespace())
	if index >= 0 {
		return errors.New("this trench is already connect")
	}
	a.trenches = append(a.trenches, trench)
	return nil
}

func (a *Ambassador) deleteTrench(trench types.Trench) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	index := a.getIndex(trench.GetName(), trench.GetNamespace())
	if index < 0 {
		return errors.New("this trench is not connected")
	}
	a.trenches = append(a.trenches[:index], a.trenches[index+1:]...)
	return trench.Delete(context.Background()) // todo: set context
}

func (a *Ambassador) getTrench(trenchName string, trenchNamespace string) types.Trench {
	a.mu.Lock()
	defer a.mu.Unlock()
	index := a.getIndex(trenchName, trenchNamespace)
	if index < 0 {
		return nil
	}
	return a.trenches[index]
}

func (a *Ambassador) getIndex(trenchName string, trenchNamespace string) int {
	for i, trench := range a.trenches {
		if trench.GetName() == trenchName && trench.GetNamespace() == trenchNamespace {
			return i
		}
	}
	return -1
}
