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

package main

import (
	"io"
	"os"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
)

// FrontendNetworkService -
// Monitor availibilty of frontends. If no feasible frontend is announced, do NOT advertise
// loadbalancer NSE to proxies. The aim is to control egress traffic flow based on frontends
// with external connectivity.
// The solution is based on NSP (maintaining information on the frontends).
//
// Currently only the composite LB-FE use case is supported:
// "NSP Target" registered by a frontend contains hostname information that is used by the
// loadbalancer to determine collocation.
// Upon events the loadbalancer registers/unregisters its NSE a meridio proxy is interested in,
// thus controlling egress traffic from proxies. While also informs SimpleNetworkService through
// serviceControlDispatcher to secure ingress LB functionality in case FE recovers.
type FrontendNetworkService struct {
	loadBalancerEndpoint                 *endpoint.Endpoint
	networkServicePlateformClient        *nsp.NetworkServicePlateformClient
	networkServicePlateformServiceStream nspAPI.NetworkServicePlateformService_MonitorClient
	myHostName                           string
	serviceControlDispatcher             *serviceControlDispatcher
}

// Start -
func (fns *FrontendNetworkService) Start() {
	var err error
	fns.networkServicePlateformServiceStream, err = fns.networkServicePlateformClient.MonitorType(nspAPI.Target_FRONTEND)
	if err != nil {
		logrus.Errorf("FrontendNetworkService: err MonitorType(%v): %v", nspAPI.Target_FRONTEND, err)
	}
	go fns.recv()
}

func (fns *FrontendNetworkService) recv() {
	for {
		targetEvent, err := fns.networkServicePlateformServiceStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("SimpleNetworkService: event err: %v", err)
			break
		}
		target := targetEvent.GetTarget()

		logrus.Debugf("FrontendNetworkService: event: %v", target)

		if fns.isLocal(target) {
			if targetEvent.Status == nspAPI.TargetEvent_Register {
				logrus.Infof("FrontendNetworkService: (local) FE available: %v", target.GetContext()[nsp.Identifier.String()])
				// inform controlled services they are allowed to operate:
				// SimpleNetworkService is allowed to accept new Targets.
				if fns.serviceControlDispatcher != nil {
					fns.serviceControlDispatcher.Dispatch(true)
				}
				// announce the southbound NSE (to the proxies, so that they could
				// establish NSM connection, and forward egress traffic to this LB)
				err := fns.loadBalancerEndpoint.Announce()
				if err != nil {
					logrus.Errorf("FrontendNetworkService: endpoint announce err: %v", err)
					continue
				}
			} else if targetEvent.Status == nspAPI.TargetEvent_Unregister {
				logrus.Warnf("FrontendNetworkService: (local) FE unavailable: %v", target.GetContext()[nsp.Identifier.String()])
				// inform controlled services they must pause operation:
				// SimpleNetworkService must not accept new Targets, and must
				// clean-up known Targets. (The nsm interfaces in ingress routes become unusable
				// once SimpleNetworkServiceClient learns the southbound NSE is removed, as it
				// closes repective NSM connection.)
				if fns.serviceControlDispatcher != nil {
					fns.serviceControlDispatcher.Dispatch(false)
				}

				// denounce southbound NSE (to the proxies, in order to block egress
				// traffic; proxies monitor the LB NSE endpoints via registry)
				err := fns.loadBalancerEndpoint.Denounce()
				if err != nil {
					logrus.Errorf("FrontendNetworkService: endpoint denounce err: %v", err)
					continue
				}
			}
		}
	}
}

// isLocal -
// Check if target is running in the same POD as the loadbalancer.
// (Targets of type FRONTEND are supposed to have their hostname as identifier.)
func (fns *FrontendNetworkService) isLocal(target *nspAPI.Target) bool {
	identifierStr, exists := target.GetContext()[nsp.Identifier.String()]
	if !exists {
		logrus.Errorf("FrontendNetworkService: identifier does not exist: %v", target.Context)
		return false
	}
	return identifierStr == fns.myHostName
}

// NewFrontendNetworkService -
func NewFrontendNetworkService(networkServicePlateformClient *nsp.NetworkServicePlateformClient, loadBalancerEndpoint *endpoint.Endpoint, serviceControlDispatcher *serviceControlDispatcher) *FrontendNetworkService {
	frontendNetworkService := &FrontendNetworkService{
		loadBalancerEndpoint:          loadBalancerEndpoint,
		networkServicePlateformClient: networkServicePlateformClient,
		serviceControlDispatcher:      serviceControlDispatcher,
	}
	frontendNetworkService.myHostName, _ = os.Hostname()
	return frontendNetworkService
}

type ServiceControl interface {
	GetServiceControlChannel() interface{}
}

type serviceControlDispatcher struct {
	mu       sync.Mutex
	handlers []ServiceControl
}

func NewServiceControlDispatcher(handlers ...ServiceControl) *serviceControlDispatcher {
	return &serviceControlDispatcher{
		handlers: handlers,
	}
}

func (scd *serviceControlDispatcher) AddService(handlers ...ServiceControl) {
	scd.mu.Lock()
	defer scd.mu.Unlock()
	// TODO: double check if already exists
	scd.handlers = append(scd.handlers, handlers...)
}

func (scd *serviceControlDispatcher) Dispatch(serviceControl interface{}) {
	scd.mu.Lock()
	defer scd.mu.Unlock()
	for _, h := range scd.handlers {
		switch event := serviceControl.(type) {
		case bool:
			h.GetServiceControlChannel().(chan<- bool) <- event
		}
	}
}
