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

package client

import (
	"context"
	"errors"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
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
		req := proto.Clone(request).(*networkservice.NetworkServiceRequest)
		connection, err := snsc.networkServiceClient.Request(context.Background(), req)
		if err != nil {
			time.Sleep(15 * time.Second)
			logrus.Errorf("Network Service Client: Request err: %v", err)
			continue
		}
		logrus.Debugf("Network Service Client: Got connection: %v", connection)
		snsc.Connection = connection

		// expiration time based on NSM updatepath (connection will be refreshed by NSM after this)
		ts := connection.GetCurrentPathSegment().GetExpires()
		if err := ts.CheckValid(); err == nil {
			expireTime := ts.AsTime()
			scale := 1. / 3.
			path := connection.GetPath()
			if len(path.PathSegments) > 1 {
				scale = 0.2 + 0.2*float64(path.Index)/float64(len(path.PathSegments))
			}
			duration := time.Duration(float64(time.Until(expireTime)) * scale)
			logrus.Debugf("Network Service Client: connection duration: %v", duration)
		}
		break
	}
	return nil
}

func (snsc *SimpleNetworkServiceClient) Close() error {
	closeCtx, cancelClose := context.WithTimeout(context.Background(), snsc.config.RequestTimeout)
	details := ""
	if snsc.Connection != nil {
		details += "id: " + snsc.Connection.GetId() + ", endpoint: " + snsc.Connection.GetNetworkServiceEndpointName()
		if snsc.Connection.GetContext() != nil && snsc.Connection.GetContext().GetIpContext() != nil {
			details += ", ips: " + snsc.Connection.GetContext().GetIpContext().String()
		}
	}

	defer func() {
		cancelClose()
		logrus.Debugf("Network Service Client: Close concluded (%v)", details)
	}()

	logrus.Debugf("Network Service Client: Close connection (%v)", details)
	_, _ = snsc.networkServiceClient.Close(closeCtx, snsc.Connection)

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
func NewSimpleNetworkServiceClient(config *Config, nsmAPIClient *nsm.APIClient, additionalFunctionality ...networkservice.NetworkServiceClient) NetworkServiceClient {
	simpleNetworkServiceClient := &SimpleNetworkServiceClient{
		config:               config,
		networkServiceClient: newClient(context.Background(), config.Name, nsmAPIClient, additionalFunctionality...),
	}

	return simpleNetworkServiceClient
}
