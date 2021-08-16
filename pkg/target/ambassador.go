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
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	targetAPI "github.com/nordix/meridio/api/target"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Ambassador struct {
	context                  context.Context
	listener                 net.Listener
	server                   *grpc.Server
	port                     int
	vips                     []string
	trenches                 []*Trench
	trenchNamespace          string
	config                   *Config
	watchConduitsSubscribers sync.Map // map[nspAPI.Ambassador_WatchConduitsServer]struct{}
	watchStreamsSubscribers  sync.Map // map[nspAPI.Ambassador_WatchStreamsServer]struct{}
	conduitWatcher           chan *ConduitEvent
	streamWatcher            chan *StreamEvent
}

func (a *Ambassador) Connect(ctx context.Context, conduit *targetAPI.Conduit) (*empty.Empty, error) {
	logrus.Infof("Connect to conduit: %v trench %v (%v)", conduit.NetworkServiceName, conduit.Trench.Name, a.trenchNamespace)
	trench := a.getTrench(conduit.Trench.Name, a.trenchNamespace)
	if trench == nil {
		trench = a.addTrench(conduit.Trench.Name, a.trenchNamespace)
	}
	_, err := trench.AddConduit(conduit.NetworkServiceName, a.conduitWatcher)
	return &empty.Empty{}, err
}

func (a *Ambassador) Disconnect(ctx context.Context, conduit *targetAPI.Conduit) (*empty.Empty, error) {
	logrus.Infof("Disconnect from conduit: %v trench %v (%v)", conduit.NetworkServiceName, conduit.Trench.Name, a.trenchNamespace)
	trench := a.getTrench(conduit.Trench.Name, a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	err := trench.DeleteConduit(conduit.NetworkServiceName)
	if err != nil {
		return &empty.Empty{}, err
	}
	err = a.deleteTrench(conduit.Trench.Name, a.trenchNamespace) // TODO
	return &empty.Empty{}, err
}

func (a *Ambassador) Request(ctx context.Context, stream *targetAPI.Stream) (*empty.Empty, error) {
	logrus.Infof("Request stream: %v trench %v (%v)", stream.Conduit.NetworkServiceName, stream.Conduit.Trench.Name, a.trenchNamespace)
	trench := a.getTrench(stream.Conduit.Trench.Name, a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	conduit := trench.GetConduit(stream.Conduit.NetworkServiceName)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	err := conduit.RequestStream(a.streamWatcher)
	return &empty.Empty{}, err
}

func (a *Ambassador) Close(ctx context.Context, stream *targetAPI.Stream) (*empty.Empty, error) {
	logrus.Infof("Close stream: %v trench %v (%v)", stream.Conduit.NetworkServiceName, stream.Conduit.Trench.Name, a.trenchNamespace)
	trench := a.getTrench(stream.Conduit.Trench.Name, a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	conduit := trench.GetConduit(stream.Conduit.NetworkServiceName)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	err := conduit.DeleteStream()
	return &empty.Empty{}, err
}

func (a *Ambassador) WatchConduits(empty *empty.Empty, stream targetAPI.Ambassador_WatchConduitsServer) error {
	a.watchConduitsSubscribers.Store(stream, struct{}{})
	for _, conduit := range a.trenches[0].conduits {
		a.notifyConduitsSubscriber(stream, conduit, Connect)
	}
	<-stream.Context().Done()
	a.watchConduitsSubscribers.Delete(stream)
	return nil
}

func (a *Ambassador) WatchStreams(empty *empty.Empty, stream targetAPI.Ambassador_WatchStreamsServer) error {
	a.watchStreamsSubscribers.Store(stream, struct{}{})
	for _, conduit := range a.trenches[0].conduits {
		a.notifyStreamsSubscriber(stream, conduit.stream, Request)
	}
	<-stream.Context().Done()
	a.watchStreamsSubscribers.Delete(stream)
	return nil
}

func (a *Ambassador) notifyConduitsSubscribers(conduit *Conduit, status ConduitStatus) {
	a.watchConduitsSubscribers.Range(func(key interface{}, value interface{}) bool {
		a.notifyConduitsSubscriber(key.(targetAPI.Ambassador_WatchConduitsServer), conduit, status)
		return true
	})
}

func (a *Ambassador) notifyConduitsSubscriber(subscriber targetAPI.Ambassador_WatchConduitsServer, conduit *Conduit, status ConduitStatus) {
	conduitEvent := &targetAPI.ConduitEvent{
		ConduitEventStatus: targetAPI.ConduitEventStatus(status),
		Conduit: &targetAPI.Conduit{
			NetworkServiceName: conduit.name,
			Trench: &targetAPI.Trench{
				Name: conduit.GetTrenchName(),
			},
		},
	}
	_ = subscriber.Send(conduitEvent)
}

func (a *Ambassador) notifyStreamsSubscribers(stream *Stream, status StreamStatus) {
	a.watchStreamsSubscribers.Range(func(key interface{}, value interface{}) bool {
		a.notifyStreamsSubscriber(key.(targetAPI.Ambassador_WatchStreamsServer), stream, status)
		return true
	})
}

func (a *Ambassador) notifyStreamsSubscriber(subscriber targetAPI.Ambassador_WatchStreamsServer, stream *Stream, status StreamStatus) {
	streamEvent := &targetAPI.StreamEvent{
		StreamEventStatus: targetAPI.StreamEventStatus(status),
		Stream: &targetAPI.Stream{
			Conduit: &targetAPI.Conduit{
				NetworkServiceName: stream.GetConduitName(),
				Trench: &targetAPI.Trench{
					Name: stream.GetTrenchName(),
				},
			},
		},
	}
	_ = subscriber.Send(streamEvent)
}

func (a *Ambassador) addTrench(name string, namespace string) *Trench {
	if len(a.trenches) >= 1 { // TODO
		return a.trenches[0]
	}
	trench := a.getTrench(name, namespace)
	if trench != nil {
		return trench
	}
	trench = NewTrench(name, namespace, a.config)
	a.trenches = append(a.trenches, trench)
	return trench
}

func (a *Ambassador) deleteTrench(name string, namespace string) error {
	for index, trench := range a.trenches {
		if trench.name == name && trench.namespace == namespace {
			a.trenches = append(a.trenches[:index], a.trenches[index+1:]...)
			return trench.Delete()
		}
	}
	return nil
}

func (a *Ambassador) getTrench(name string, namespace string) *Trench {
	for _, trench := range a.trenches {
		if trench.name == name && trench.namespace == namespace {
			return trench
		}
	}
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
		err := trench.Delete()
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
			a.notifyConduitsSubscribers(conduitEvent.Conduit, conduitEvent.ConduitStatus)
		case streamEvent := <-a.streamWatcher:
			a.notifyStreamsSubscribers(streamEvent.Stream, streamEvent.StreamStatus)
		case <-a.context.Done():
			return
		}
	}
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
		trenches:        []*Trench{},
		trenchNamespace: trenchNamespace,
		config:          config,
		conduitWatcher:  make(chan *ConduitEvent, 10),
		streamWatcher:   make(chan *StreamEvent, 10),
	}

	targetAPI.RegisterAmbassadorServer(s, ambassador)

	return ambassador, nil
}
