package target

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	targetAPI "github.com/nordix/meridio/api/target"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Ambassador struct {
	listener      net.Listener
	server        *grpc.Server
	port          int
	defaultTrench string
	vips          []string
	trenches      []*Trench
	config        *Config
}

func (a *Ambassador) Connect(ctx context.Context, conduit *targetAPI.Conduit) (*empty.Empty, error) {
	trenchName := conduit.Trench
	if trenchName == "" {
		trenchName = a.defaultTrench
	}
	logrus.Infof("Connect to conduit: %v trench %v", conduit.NetworkServiceName, trenchName)
	trench := a.getTrench(trenchName)
	if trench == nil {
		trench = a.addTrench(trenchName)
	}
	_, err := trench.AddConduit(conduit.NetworkServiceName)
	return &empty.Empty{}, err
}

func (a *Ambassador) Disconnect(ctx context.Context, conduit *targetAPI.Conduit) (*empty.Empty, error) {
	trenchName := conduit.Trench
	if trenchName == "" {
		trenchName = a.defaultTrench
	}
	logrus.Infof("Disconnect from conduit: %v trench %v", conduit.NetworkServiceName, trenchName)
	trench := a.getTrench(trenchName)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	err := trench.DeleteConduit(conduit.NetworkServiceName)
	if err != nil {
		return &empty.Empty{}, err
	}
	err = a.deleteTrench(trenchName) // TODO
	return &empty.Empty{}, err
}

func (a *Ambassador) Request(ctx context.Context, stream *targetAPI.Stream) (*empty.Empty, error) {
	trenchName := stream.Conduit.Trench
	if trenchName == "" {
		trenchName = a.defaultTrench
	}
	logrus.Infof("Request stream: %v trench %v", stream.Conduit.NetworkServiceName, trenchName)
	trench := a.getTrench(trenchName)
	if trench == nil {
		return &empty.Empty{}, nil
	}
	conduit := trench.GetConduit(stream.Conduit.NetworkServiceName)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	err := conduit.RequestStream()
	return &empty.Empty{}, err
}

func (a *Ambassador) Close(ctx context.Context, stream *targetAPI.Stream) (*empty.Empty, error) {
	trenchName := stream.Conduit.Trench
	if trenchName == "" {
		trenchName = a.defaultTrench
	}
	logrus.Infof("Close stream: %v trench %v", stream.Conduit.NetworkServiceName, trenchName)
	trench := a.getTrench(trenchName)
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

func (a *Ambassador) addTrench(name string) *Trench {
	if len(a.trenches) >= 1 { // TODO
		return a.trenches[0]
	}
	trench := a.getTrench(name)
	if trench != nil {
		return trench
	}
	trench = NewTrench(name, a.config)
	a.trenches = append(a.trenches, trench)
	return trench
}

func (a *Ambassador) deleteTrench(name string) error {
	for index, trench := range a.trenches {
		if trench.name == name {
			a.trenches = append(a.trenches[:index], a.trenches[index+1:]...)
			return trench.Delete()
		}
	}
	return nil
}

func (a *Ambassador) getTrench(name string) *Trench {
	for _, trench := range a.trenches {
		if trench.name == name {
			return trench
		}
	}
	return nil
}

func (a *Ambassador) Start(ctx context.Context) error {
	a.config.apiClient = nsm.NewAPIClient(ctx, a.config.nsmConfig)
	return a.server.Serve(a.listener)
}

func (a *Ambassador) Delete() error {
	a.server.Stop()
	for _, trench := range a.trenches {
		err := trench.Delete()
		if err != nil {
			logrus.Errorf("Error deleting a trench: %v", err)
		}
	}
	return nil
}

func NewAmbassador(port int, trench string, config *Config) (*Ambassador, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()

	ambassador := &Ambassador{
		listener:      lis,
		server:        s,
		port:          port,
		defaultTrench: trench,
		vips:          []string{},
		trenches:      []*Trench{},
		config:        config,
	}

	targetAPI.RegisterAmbassadorServer(s, ambassador)

	return ambassador, nil
}
