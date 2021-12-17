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
	"log"
	"net"
	"os"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	targetAPI "github.com/nordix/meridio/api/target/v1"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/target/conduit"
	"github.com/nordix/meridio/pkg/target/stream"
	"github.com/nordix/meridio/pkg/target/trench"
	"github.com/nordix/meridio/pkg/target/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Ambassador struct {
	context                  context.Context
	listener                 net.Listener
	server                   *grpc.Server
	socket                   string
	vips                     []string
	trenches                 []types.Trench
	trenchNamespace          string
	config                   *Config
	watchConduitsSubscribers sync.Map // map[targetAPI.Ambassador_WatchConduitServer]*conduitWatcher
	watchStreamsSubscribers  sync.Map // map[targetAPI.Ambassador_WatchStreamServer]*streamWatcher
	evenChan                 chan struct{}
	mu                       sync.Mutex
}

func NewAmbassador(socket string, trenchNamespace string, config *Config) (*Ambassador, error) {
	if err := os.RemoveAll(socket); err != nil {
		log.Fatal(err)
	}
	lis, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()

	ambassador := &Ambassador{
		listener:        lis,
		server:          s,
		socket:          socket,
		vips:            []string{},
		trenches:        []types.Trench{},
		trenchNamespace: trenchNamespace,
		config:          config,
		evenChan:        make(chan struct{}, 10),
	}

	targetAPI.RegisterAmbassadorServer(s, ambassador)
	logrus.Debugf("Creating ambassador grpc health server")
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)

	return ambassador, nil
}

func (a *Ambassador) Connect(ctx context.Context, c *nspAPI.Conduit) (*empty.Empty, error) {
	logrus.Infof("Connect to conduit: %v ; trench %v (%v)", c.GetName(), c.GetTrench().GetName(), a.trenchNamespace)
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
			if errDelete != nil {
				return &empty.Empty{}, fmt.Errorf("%w; %v", err, errDelete)
			}
			return &empty.Empty{}, err
		}
	}
	_, err = conduit.New(ctx, c.GetName(), t, a.config.nodeName, a.config.apiClient, a.config.nsmConfig, a.evenChan, a.config.netUtils) // todo: api
	return &empty.Empty{}, err
}

func (a *Ambassador) Disconnect(ctx context.Context, c *nspAPI.Conduit) (*empty.Empty, error) {
	logrus.Infof("Disconnect from conduit: %v ; trench %v (%v)", c.GetName(), c.GetTrench().GetName(), a.trenchNamespace)
	trench := a.getTrench(c.GetTrench().GetName(), a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, errors.New("not connected to the trench")
	}
	conduits := trench.GetConduits(c)
	if conduits == nil || len(conduits) <= 0 || len(conduits) > 1 {
		return &empty.Empty{}, errors.New("conduit not found")
	}
	conduit := conduits[0]
	err := trench.RemoveConduit(ctx, conduit)
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, a.deleteTrench(trench) // todo
}

func (a *Ambassador) Open(ctx context.Context, s *nspAPI.Stream) (*empty.Empty, error) {
	logrus.Infof("Open stream: %v ; conduit: %v ; trench %v (%v)", s.GetName(), s.GetConduit().GetName(), s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	trench := a.getTrench(s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, errors.New("not connected to the trench")
	}
	conduits := trench.GetConduits(s.GetConduit())
	if conduits == nil || len(conduits) <= 0 || len(conduits) > 1 {
		return &empty.Empty{}, errors.New("conduit not found")
	}
	conduit := conduits[0]
	_, err := stream.New(ctx, s.GetName(), conduit, a.evenChan) // todo: api
	return &empty.Empty{}, err
}

func (a *Ambassador) Close(ctx context.Context, s *nspAPI.Stream) (*empty.Empty, error) {
	logrus.Infof("Close stream: %v ; conduit: %v ; trench %v (%v)", s.GetName(), s.GetConduit().GetName(), s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	trench := a.getTrench(s.GetConduit().GetTrench().GetName(), a.trenchNamespace)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	conduits := trench.GetConduits(s.GetConduit())
	if conduits == nil || len(conduits) <= 0 || len(conduits) > 1 {
		return &empty.Empty{}, errors.New("conduit not found")
	}
	conduit := conduits[0]
	streams := conduit.GetStreams(s)
	if streams == nil || len(streams) <= 0 || len(streams) > 1 {
		return &empty.Empty{}, errors.New("stream not found")
	}
	stream := streams[0]
	return &empty.Empty{}, conduit.RemoveStream(ctx, stream)
}

func (a *Ambassador) WatchConduit(conduitToWatch *nspAPI.Conduit, watcher targetAPI.Ambassador_WatchConduitServer) error {
	conduitWatcher := &conduitWatcher{
		watcher:        watcher,
		conduitToWatch: conduitToWatch,
	}
	a.watchConduitsSubscribers.Store(watcher, conduitWatcher)
	conduitWatcher.notify(a.getCurrentTrench())
	<-watcher.Context().Done()
	a.watchConduitsSubscribers.Delete(watcher)
	return nil
}

func (a *Ambassador) WatchStream(streamToWatch *nspAPI.Stream, watcher targetAPI.Ambassador_WatchStreamServer) error {
	streamWatcher := &streamWatcher{
		watcher:       watcher,
		streamToWatch: streamToWatch,
	}
	a.watchStreamsSubscribers.Store(watcher, streamWatcher)
	streamWatcher.notify(a.getCurrentTrench())
	<-watcher.Context().Done()
	a.watchStreamsSubscribers.Delete(watcher)
	return nil
}

func (a *Ambassador) Start(ctx context.Context) error {
	a.context = ctx
	a.config.apiClient = nsm.NewAPIClient(a.context, a.config.nsmConfig)
	go a.watcher()
	logrus.Debugf("Starting ambassador server")
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
		case <-a.evenChan:
			a.notifyConduitSubscribers()
			a.notifyStreamSubscribers()
		case <-a.context.Done():
			return
		}
	}
}

func (a *Ambassador) notifyConduitSubscribers() {
	a.watchConduitsSubscribers.Range(func(key interface{}, value interface{}) bool {
		value.(*conduitWatcher).notify(a.getCurrentTrench()) // todo
		return true
	})
}

func (a *Ambassador) notifyStreamSubscribers() {
	a.watchStreamsSubscribers.Range(func(key interface{}, value interface{}) bool {
		value.(*streamWatcher).notify(a.getCurrentTrench()) // todo
		return true
	})
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

func (a *Ambassador) getCurrentTrench() types.Trench {
	if len(a.trenches) <= 0 {
		return nil
	}
	return a.trenches[0]
}
