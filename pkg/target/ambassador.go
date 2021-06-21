package target

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	targetAPI "github.com/nordix/meridio/api/target"
	"github.com/nordix/meridio/pkg/client"
	"github.com/nordix/meridio/pkg/configuration"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Ambassador struct {
	listener      net.Listener
	server        *grpc.Server
	port          int
	trench        string
	vips          []string
	apiClient     *nsm.APIClient
	conduits      []*Conduit
	configWatcher <-chan *configuration.Config
	config        *Config
}

func (a *Ambassador) Connect(ctx context.Context, conduit *targetAPI.Conduit) (*empty.Empty, error) {
	trench := conduit.Trench
	if trench == "" {
		trench = a.trench
	}
	logrus.Infof("Connect to conduit: %v trench %v", conduit.NetworkServiceName, trench)
	a.addConduit(conduit.NetworkServiceName, trench)
	return &empty.Empty{}, nil
}

func (a *Ambassador) Disconnect(ctx context.Context, conduit *targetAPI.Conduit) (*empty.Empty, error) {
	trench := conduit.Trench
	if trench == "" {
		trench = a.trench
	}
	logrus.Infof("Disconnect from conduit: %v trench %v", conduit.NetworkServiceName, trench)
	a.deleteConduit(conduit.NetworkServiceName, trench)
	return &empty.Empty{}, nil
}

func (a *Ambassador) Request(ctx context.Context, stream *targetAPI.Stream) (*empty.Empty, error) {
	trench := stream.Conduit.Trench
	if trench == "" {
		trench = a.trench
	}
	logrus.Infof("Request stream: %v trench %v", stream.Conduit.NetworkServiceName, trench)
	conduit := a.getConduit(stream.Conduit.NetworkServiceName, trench)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	err := conduit.RequestStream()
	return &empty.Empty{}, err
}

func (a *Ambassador) Close(ctx context.Context, stream *targetAPI.Stream) (*empty.Empty, error) {
	trench := stream.Conduit.Trench
	if trench == "" {
		trench = a.trench
	}
	logrus.Infof("Close stream: %v trench %v", stream.Conduit.NetworkServiceName, trench)
	conduit := a.getConduit(stream.Conduit.NetworkServiceName, trench)
	if conduit == nil {
		return &empty.Empty{}, nil
	}
	err := conduit.CloseStream()
	return &empty.Empty{}, err
}

func (a *Ambassador) getConduit(networkServiceName string, trench string) *Conduit {
	for _, conduit := range a.conduits {
		if conduit.networkServiceName == networkServiceName && conduit.trench == trench {
			return conduit
		}
	}
	return nil
}

func (a *Ambassador) addConduit(networkServiceName string, trench string) {
	conduit := NewConduit(networkServiceName, trench, a.config)
	conduit.SetVIPs(a.vips)
	a.conduits = append(a.conduits, conduit)
	clientConfig := &client.Config{
		Name:           a.config.nsmConfig.Name,
		RequestTimeout: a.config.nsmConfig.RequestTimeout,
	}
	conduit.Request(a.apiClient.GRPCClient, clientConfig)
}

func (a *Ambassador) deleteConduit(networkServiceName string, trench string) {
	for index, conduit := range a.conduits {
		if conduit.networkServiceName == networkServiceName && conduit.trench == trench {
			a.conduits = append(a.conduits[:index], a.conduits[index+1:]...)
			conduit.Close()
			break
		}
	}
}

func (a *Ambassador) serve() {
	err := a.server.Serve(a.listener)
	if err != nil {
		logrus.Errorf("Err serve: %v", err)
	}
}

func (a *Ambassador) Start(ctx context.Context) error {
	a.apiClient = nsm.NewAPIClient(ctx, a.config.nsmConfig)
	go a.serve()

	for {
		select {
		case config := <-a.configWatcher:
			a.vips = config.VIPs
			for _, conduit := range a.conduits {
				conduit.SetVIPs(config.VIPs)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Ambassador) Delete() {
	a.server.Stop()
	for _, conduit := range a.conduits {
		a.deleteConduit(conduit.networkServiceName, conduit.trench)
	}
}

func NewAmbassador(port int, trench string, configWatcher <-chan *configuration.Config, config *Config) (*Ambassador, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()

	ambassador := &Ambassador{
		listener:      lis,
		server:        s,
		port:          port,
		trench:        trench,
		vips:          []string{},
		conduits:      []*Conduit{},
		config:        config,
		configWatcher: configWatcher,
	}

	targetAPI.RegisterAmbassadorServer(s, ambassador)

	return ambassador, nil
}
