package client

import (
	"context"
	"errors"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type SimpleNetworkServiceClient struct {
	networkServiceClient networkservice.NetworkServiceClient
	config               *Config
	Connection           *networkservice.Connection
}

func (snsc *SimpleNetworkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !snsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}
	for {
		connection, err := snsc.networkServiceClient.Request(context.Background(), request)
		if err != nil {
			time.Sleep(15 * time.Second)
			logrus.Errorf("Network Service Client: Request err: %v", err)
			continue
		}
		snsc.Connection = connection
		break
	}
	return nil
}

func (snsc *SimpleNetworkServiceClient) Close() error {
	for {
		_, err := snsc.networkServiceClient.Close(context.Background(), snsc.Connection)
		if err != nil {
			time.Sleep(15 * time.Second)
			logrus.Errorf("Network Service Client: Request err: %v", err)
			continue
		}
		break
	}
	return nil
}

func (snsc *SimpleNetworkServiceClient) requestIsValid(request *networkservice.NetworkServiceRequest) bool {
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
func NewSimpleNetworkServiceClient(config *Config, cc grpc.ClientConnInterface, additionalFunctionality ...networkservice.NetworkServiceClient) NetworkServiceClient {
	simpleNetworkServiceClient := &SimpleNetworkServiceClient{
		config:               config,
		networkServiceClient: newClient(context.Background(), config.Name, cc, additionalFunctionality...),
	}

	return simpleNetworkServiceClient
}
