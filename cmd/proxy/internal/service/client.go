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

package service

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	vfiomech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/mechanisms/vfio"
	sriovtoken "github.com/networkservicemesh/sdk-sriov/pkg/networkservice/common/token"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/nordix/meridio/cmd/proxy/internal/client"
	"github.com/nordix/meridio/cmd/proxy/internal/config"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/nsm"
	"github.com/nordix/meridio/pkg/nsm/fullmeshtracker"
	"github.com/nordix/meridio/pkg/nsm/ipcontext"
	"github.com/nordix/meridio/pkg/nsm/mtu"
	"github.com/nordix/meridio/pkg/proxy"
	proxyHealth "github.com/nordix/meridio/pkg/proxy/health"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetNSC(ctx context.Context,
	config *config.Config,
	nsmAPIClient *nsm.APIClient,
	p *proxy.Proxy,
	interfaceMonitorClient networkservice.NetworkServiceClient) client.NetworkServiceClient {

	logger := log.FromContextOrGlobal(ctx).WithValues("func", "GetNSC")
	logger.Info("Create Full Mesh NSC")
	clientConfig := &client.Config{
		Name:           config.Name,
		RequestTimeout: config.RequestTimeout,
		ConnectTo:      config.ConnectTo,
		APIClient:      nsmAPIClient,
	}
	// Note: naming the interface is left to NSM (refer to getNameFromConnection())
	// However NSM does not seem to ensure uniqueness either. Might need to revisit...
	additionalFunctionality := chain.NewNetworkServiceClient(
		sriovtoken.NewClient(),
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			vfiomech.MECHANISM:   chain.NewNetworkServiceClient(vfio.NewClient()),
			kernelmech.MECHANISM: chain.NewNetworkServiceClient(kernel.NewClient()),
		}),
		ipcontext.NewClient(p),
		interfaceMonitorClient,
		mtu.NewMtuClient(uint32(config.MTU)),
		proxyHealth.NewClient(),
		fullmeshtracker.NewClient(),
	)
	fullMeshClient := client.NewFullMeshNetworkServiceClient(ctx, clientConfig, additionalFunctionality)

	return fullMeshClient
}

func StartNSC(fullMeshClient client.NetworkServiceClient, networkServiceName string) {
	logger := log.Logger.WithValues("func", "StartNSC", "service", networkServiceName)
	logger.Info("Start Full Mesh NSC")
	err := fullMeshClient.Request(&networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: networkServiceName,
			Payload:        payload.Ethernet,
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:  cls.LOCAL,
				Type: kernelmech.MECHANISM,
			},
		},
	})
	if err != nil && status.Code(err) != codes.Canceled {
		logger.Error(err, "Full Mesh Client Request failed")
	}
	logger.Info("Full Mesh NSC stopped")
}
