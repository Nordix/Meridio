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

package service

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/noop"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	endpointOld "github.com/nordix/meridio/pkg/endpoint"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/endpoint"
	"github.com/nordix/meridio/pkg/nsm/ipcontext"
	"github.com/nordix/meridio/pkg/nsm/mtu"
	"github.com/nordix/meridio/pkg/nsm/service"
	"github.com/nordix/meridio/pkg/proxy"
)

func StartNSE(ctx context.Context,
	config *endpointOld.Config,
	nsmAPIClient *nsm.APIClient,
	p *proxy.Proxy,
	interfaceMonitorServer networkservice.NetworkServiceServer) *endpoint.Endpoint {

	logger := log.FromContextOrGlobal(ctx)
	logger.Info("Start NSE")
	additionalFunctionality := []networkservice.NetworkServiceServer{
		// Note: naming the interface is left to NSM (refer to getNameFromConnection())
		// However NSM does not seem to ensure uniqueness either. Might need to revisit...
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: kernel.NewServer(),
			noop.MECHANISM:       null.NewServer(),
		}),
		ipcontext.NewServer(p),
		interfaceMonitorServer,
		mtu.NewMtuServer(uint32(config.MTU)),
		sendfd.NewServer(),
	}

	ns := &registry.NetworkService{
		Name:    config.ServiceName,
		Payload: payload.Ethernet,
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{
					"nodeName": "{{.nodeName}}",
				},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"nodeName": "{{.nodeName}}",
						},
					},
				},
			},
		},
	}
	logger.V(1).Info("Create NS", "ns", ns)

	service := service.New(nsmAPIClient.NetworkServiceRegistryClient, ns)
	err := service.Register(ctx)
	if err != nil {
		log.Fatal(logger, "Registering NS", "error", err)
	}

	nse := &registry.NetworkServiceEndpoint{
		Name:                config.Name,
		NetworkServiceNames: []string{config.ServiceName},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			config.ServiceName: {
				Labels: config.Labels,
			},
		},
	}
	logger.V(1).Info("Create NSE", "nse", nse)

	endpoint, err := endpoint.New(config.MaxTokenLifetime,
		nsmAPIClient.NetworkServiceEndpointRegistryClient,
		nse,
		additionalFunctionality...)
	if err != nil {
		log.Fatal(logger, "Creating NSE", "error", err)
	}
	err = endpoint.Register(ctx)
	if err != nil {
		log.Fatal(logger, "Registering NSE", "error", err)
	}
	return endpoint
}
