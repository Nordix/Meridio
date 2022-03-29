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

package endpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/sirupsen/logrus"
)

type Endpoint struct {
	NSERegistryClient registry.NetworkServiceEndpointRegistryClient
	NSE               *registry.NetworkServiceEndpoint
	Server            NetworkServiceEndpointServer
}

type NetworkServiceEndpointServer interface {
	Start(ctx context.Context) error
	Stop() error
	GetUrl() string
}

func New(maxTokenLifetime time.Duration,
	nseRegistryClient registry.NetworkServiceEndpointRegistryClient,
	nse *registry.NetworkServiceEndpoint,
	additionalFunctionality ...networkservice.NetworkServiceServer) (*Endpoint, error) {
	s := NewServer(nse.Name, maxTokenLifetime, additionalFunctionality)
	err := s.Start(context.Background())
	if err != nil {
		err = fmt.Errorf("err starting the NSE server: %v", err)
		errStop := s.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%v ; Err stopping the NSE server: %v", err, errStop)
		}
		return nil, err
	}
	e := &Endpoint{
		NSERegistryClient: nseRegistryClient,
		NSE:               nse,
		Server:            s,
	}
	return e, nil
}

func (e *Endpoint) Delete(ctx context.Context) error {
	logrus.Infof("Endpoint Delete")
	var errFinal error
	err := e.Unregister(ctx)
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	err = e.Server.Stop()
	if err != nil {
		errFinal = fmt.Errorf("%w; %v", errFinal, err)
	}
	return errFinal
}

func (e *Endpoint) Register(ctx context.Context) error {
	e.NSE.Url = e.Server.GetUrl()
	e.NSE.ExpirationTime = nil
	var err error
	e.NSE, err = e.NSERegistryClient.Register(ctx, e.NSE)
	return err
}

func (e *Endpoint) Unregister(ctx context.Context) error {
	e.NSE.ExpirationTime = &timestamp.Timestamp{
		Seconds: -1,
	}
	_, err := e.NSERegistryClient.Unregister(ctx, e.NSE)
	return err
}
