/*
Copyright (c) 2021-2022 Nordix Foundation

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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/retry"
	"github.com/sirupsen/logrus"
)

const (
	retryInterval = 10 * time.Second
)

type SimpleNetworkServiceClient struct {
	networkServiceClient networkservice.NetworkServiceClient
	config               *Config
	connection           *networkservice.Connection
	ctx                  context.Context
	requestCtx           context.Context
	requestCancel        context.CancelFunc
	mu                   sync.Mutex
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

	resp, err := snsc.networkServiceClient.Request(snsc.requestCtx, request)
	if err != nil {
		return err
	}
	logrus.Debugf("Network Service Client: Got connection: %v", resp)
	snsc.connection = resp
	snsc.printConnectionExpTime()

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
		// close established network service connection
		ctx := snsc.ctx

		if ctx.Err() != nil {
			logrus.Tracef("Network Service Client: Close with new context, old not usable (%v)", ctx.Err())
			ctx = context.Background()
		}

		// Note: nsm retry client keeps trying to close connection until either succeeds or the passed context is done
		ctx, cancel := context.WithTimeout(ctx, snsc.config.RequestTimeout)

		details := fmt.Sprintf("endpoint: %s, service: %s, id: %s",
			snsc.connection.GetNetworkServiceEndpointName(), snsc.connection.GetNetworkService(), snsc.connection.GetId())
		if snsc.connection.GetContext() != nil && snsc.connection.GetContext().GetIpContext() != nil {
			details += fmt.Sprintf(" ips: %s", snsc.connection.GetContext().GetIpContext().String())
		}

		defer func() {
			cancel()
			logrus.Debugf("Network Service Client: Close concluded (%v)", details)
		}()

		logrus.Debugf("Network Service Client: Close connection (%v)", details)
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

// printConnectionExpTime -
// Prints expiration time of established connection for debugging purposes
func (snsc *SimpleNetworkServiceClient) printConnectionExpTime() {
	connection := snsc.connection

	// expiration time based on the local path segment
	ts := connection.GetCurrentPathSegment().GetExpires()
	if err := ts.CheckValid(); err == nil {
		expireTime := ts.AsTime()
		scale := 1. / 3.
		path := connection.GetPath()
		if len(path.PathSegments) > 1 {
			scale = 0.2 + 0.2*float64(path.Index)/float64(len(path.PathSegments))
		}
		duration := time.Duration(float64(time.Until(expireTime)) * scale)
		logrus.Debugf("Network Service Client: connection duration (local): %v", duration)
	}

	// expiration time based on NSM@8e96470 updatepath (considers all path segments)
	{
		var minTimeout *time.Duration
		var expireTime time.Time
		for _, segment := range connection.GetPath().GetPathSegments() {
			ts := segment.GetExpires()
			if err := ts.CheckValid(); err != nil {
				break
			}
			expTime := ts.AsTime()
			timeout := time.Until(expTime)

			if minTimeout == nil || timeout < *minTimeout {
				if minTimeout == nil {
					minTimeout = new(time.Duration)
				}

				*minTimeout = timeout
				expireTime = expTime
			}
		}
		if minTimeout != nil {
			logrus.Debugf("expiration after %s at %s", minTimeout.String(), expireTime.UTC())
		}
		if minTimeout != nil && *minTimeout > 0 {
			scale := 1. / 3.
			path := connection.GetPath()
			if len(path.PathSegments) > 1 {
				scale = 0.2 + 0.2*float64(path.Index)/float64(len(path.PathSegments))
			}
			duration := time.Duration(float64(*minTimeout) * scale)
			logrus.Debugf("Network Service Client: connection duration (end-to-end): %v", duration)
		}
	}
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
	}

	return simpleNetworkServiceClient
}
