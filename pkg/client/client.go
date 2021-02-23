package client

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
)

type NetworkServiceClient struct {
	Id                         string
	NetworkServiceName         string
	NetworkServiceEndpointName string
	Labels                     map[string]string
	Connection                 *networkservice.Connection
	nsmgrClient                NSMgrClient
}

type NSMgrClient interface {
	Request(*networkservice.NetworkServiceRequest) (*networkservice.Connection, error)
	Close(*networkservice.Connection) (*empty.Empty, error)
}

// Request
func (nsc *NetworkServiceClient) Request() {
	request := nsc.prepareRequest()
	for true {
		var err error
		nsc.Connection, err = nsc.nsmgrClient.Request(request)
		if err != nil {
			time.Sleep(2 * time.Second)
			logrus.Errorf("Full Mesh Client: NetworkServiceClient Request err: %v", err)
			continue
		}
		break
	}
}

// Close -
func (nsc *NetworkServiceClient) Close() {
	var err error
	_, err = nsc.nsmgrClient.Close(nsc.Connection)
	if err != nil {
		time.Sleep(2 * time.Second)
		logrus.Errorf("Full Mesh Client: NetworkServiceClient Close err: %v", err)
	}
}

func (nsc *NetworkServiceClient) prepareRequest() *networkservice.NetworkServiceRequest {
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:                         nsc.Id,
			NetworkService:             nsc.NetworkServiceName,
			Labels:                     nsc.Labels,
			NetworkServiceEndpointName: nsc.NetworkServiceEndpointName,
		},
	}
	return request
}

// NewnetworkServiceClient
func NewNetworkServiceClient(networkServiceName string, nsmgrClient NSMgrClient) *NetworkServiceClient {
	identifier := rand.Intn(100)
	id := fmt.Sprintf("%d", identifier)

	networkServiceClient := &NetworkServiceClient{
		Id:                 id,
		NetworkServiceName: networkServiceName,
		nsmgrClient:        nsmgrClient,
	}

	return networkServiceClient
}
