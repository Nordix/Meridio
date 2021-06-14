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
	"github.com/nordix/meridio/pkg/networking"
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
	nsmConfig     *nsm.Config
	connections   []*Connection
	configWatcher <-chan *configuration.Config
	netUtils      networking.Utils
}

func (a *Ambassador) Request(ctx context.Context, connection *targetAPI.Connection) (*empty.Empty, error) {
	a.addConnection(connection.NetworkServiceName)
	return &empty.Empty{}, nil
}

func (a *Ambassador) Close(ctx context.Context, connection *targetAPI.Connection) (*empty.Empty, error) {
	a.deleteConnection(connection.NetworkServiceName)
	return &empty.Empty{}, nil
}

func (a *Ambassador) addConnection(networkServiceName string) {
	connection := NewConnection(networkServiceName, a.trench, a.netUtils)
	connection.SetVIPs(a.vips)
	a.connections = append(a.connections, connection)
	clientConfig := &client.Config{
		Name:           a.nsmConfig.Name,
		RequestTimeout: a.nsmConfig.RequestTimeout,
	}
	connection.Request(a.apiClient.GRPCClient, clientConfig)
}

func (a *Ambassador) deleteConnection(networkServiceName string) {
	for index, connection := range a.connections {
		if connection.networkServiceName == networkServiceName {
			a.connections = append(a.connections[:index], a.connections[index+1:]...)
			connection.Close()
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
	a.apiClient = nsm.NewAPIClient(ctx, a.nsmConfig)
	go a.serve()

	for {
		select {
		case config := <-a.configWatcher:
			a.vips = config.VIPs
			for _, connection := range a.connections {
				connection.SetVIPs(config.VIPs)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Ambassador) Stop() {
}

func NewAmbassador(port int, trench string, nsmConfig *nsm.Config, configWatcher <-chan *configuration.Config, netUtils networking.Utils) (*Ambassador, error) {
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
		connections:   []*Connection{},
		nsmConfig:     nsmConfig,
		configWatcher: configWatcher,
		netUtils:      netUtils,
	}

	targetAPI.RegisterAmbassadorServer(s, ambassador)

	return ambassador, nil
}
