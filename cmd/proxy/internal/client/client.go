/*
Copyright (c) 2021-2023 Nordix Foundation

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
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/retry"
	"github.com/nordix/meridio/pkg/log"
)

const (
	retryInterval = 10 * time.Second
)

// TODO: Consider removing SimpleNetworkServiceClient and replacing it with an nsm retry client in fullMeshClient.
type SimpleNetworkServiceClient struct {
	networkServiceClient networkservice.NetworkServiceClient
	config               *Config
	connection           *networkservice.Connection
	ctx                  context.Context
	requestCtx           context.Context
	requestCancel        context.CancelFunc
	mu                   sync.Mutex
	logger               logr.Logger
}

// Request -
// SimpleNetworkServiceClient is a retry.Client.
// Thus Request() blocks as it keeps trying to establish the connection, while the request
// is cloned at each attempt.
func (snsc *SimpleNetworkServiceClient) Request(request *networkservice.NetworkServiceRequest) error {
	if !snsc.requestIsValid(request) {
		return errors.New("request is not valid")
	}

	snsc.mu.Lock()
	defer snsc.mu.Unlock()

	logger := snsc.logger.WithValues("func", "Request")
	resp, err := snsc.networkServiceClient.Request(snsc.requestCtx, request)
	if err != nil {
		return fmt.Errorf("network service client failed to request connection: %w", err)
	}
	logger.V(1).Info("Connected", "connection", resp)
	snsc.connection = resp

	return nil
}

// Close -
// Closes established nsm connection or cancels pending Request()
//
// Note:
// The context passed when creating SimpleNetworkServiceClient might be closed by the time
// Close() gets called. Therefore check if context is still usable, otherwise use background context.
func (snsc *SimpleNetworkServiceClient) Close() error {
	if snsc.requestCancel != nil {
		// first cancel any pending Request()
		snsc.requestCancel()
	}

	snsc.mu.Lock()
	defer snsc.mu.Unlock()

	if snsc.connection != nil {
		logger := snsc.logger.WithValues("func", "Close",
			"name", snsc.connection.GetNetworkServiceEndpointName(),
			"service", snsc.connection.GetNetworkService(),
			"id", snsc.connection.GetId(),
			"connection context", snsc.connection.GetContext(),
		)
		// close established network service connection
		ctx := snsc.ctx
		if ctx.Err() != nil {
			logger.V(2).Info("Close connection using new context", "error", ctx.Err())
			ctx = context.Background()
		}
		// Note: nsm retry client keeps trying to close connection until either succeeds or the passed context is done
		ctx, cancel := context.WithTimeout(ctx, snsc.config.RequestTimeout)
		defer func() {
			cancel()
			logger.V(1).Info("Connection close concluded")
		}()
		logger.V(1).Info("Close connection")
		_, _ = snsc.networkServiceClient.Close(ctx, snsc.connection)
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

// NewSimpleNetworkServiceClient -
// Wwraps NetworkServiceClient using nsm retry.Client
func NewSimpleNetworkServiceClient(ctx context.Context, config *Config, client networkservice.NetworkServiceClient) NetworkServiceClient {
	//client := newClient(ctx, config.Name, config.APIClient, config.AdditionalFunctionality)
	cancelCtx, cancel := context.WithCancel(ctx)

	simpleNetworkServiceClient := &SimpleNetworkServiceClient{
		config:               config,
		networkServiceClient: retry.NewClient(client, retry.WithTryTimeout(config.RequestTimeout), retry.WithInterval(retryInterval)),
		ctx:                  ctx,
		requestCtx:           cancelCtx,
		requestCancel:        cancel,
		logger:               log.FromContextOrGlobal(ctx).WithValues("class", "SimpleNetworkServiceClient"),
	}

	return simpleNetworkServiceClient
}
