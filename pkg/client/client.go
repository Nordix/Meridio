package client

import (
	"context"
	"errors"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type networkServiceClient struct {
	networkServiceClient networkservice.NetworkServiceClient
	config               *Config
	connection           *networkservice.Connection
}

func (nsc *networkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !nsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}
	for {
		connection, err := nsc.networkServiceClient.Request(context.Background(), request)
		if err != nil {
			time.Sleep(15 * time.Second)
			logrus.Errorf("Network Service Client: Request err: %v", err)
			continue
		}
		nsc.connection = connection
		break
	}
	return nil
}

func (nsc *networkServiceClient) Close() error {
	for {
		_, err := nsc.networkServiceClient.Close(context.Background(), nsc.connection)
		if err != nil {
			time.Sleep(15 * time.Second)
			logrus.Errorf("Network Service Client: Request err: %v", err)
			continue
		}
		break
	}
	return nil
}

func (nsc *networkServiceClient) requestIsValid(request *networkservice.NetworkServiceRequest) bool {
	if request == nil {
		return false
	}
	if request.GetMechanismPreferences() == nil || len(request.GetMechanismPreferences()) == 0 {
		return false
	}
	if request.GetConnection() == nil || request.GetConnection().NetworkService == "" {
		return false
	}
	return true
}

// NewnetworkServiceClient
func NewNetworkServiceClient(config *Config, cc grpc.ClientConnInterface, additionalFunctionality ...networkservice.NetworkServiceClient) NetworkServiceClient {
	networkServiceClient := &networkServiceClient{
		config:               config,
		networkServiceClient: newClient(context.Background(), config.Name, cc, additionalFunctionality...),
	}

	return networkServiceClient
}
