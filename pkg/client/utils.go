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

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/nordix/meridio/pkg/nsm"
)

func expirationTimeIsNull(expirationTime *timestamp.Timestamp) bool {
	nullTImeStamp := &timestamp.Timestamp{
		Seconds: -1,
	}
	return expirationTime == nil || expirationTime.AsTime().Equal(nullTImeStamp.AsTime())
}

// newClient -
// Creates networkservice.NetworkServiceClient relying on NSM's client.NewClient API
//
// Note:
// Refresh Client comes from the NSM sdk version used. In case of NSM 1.1.1 the built-in
// refresh might lead to connection issues if the different path segments have different
// maxTokenLifetime configured (unless the NSC side has the lowest maxtokenlifetime).
// To load a custom refresh client the client.NewClient chain must be replaced by the
// chain in the comment with the desired refresh (a more up to date backport of the NSM
// refresh client is avaialbe in Meridio/pkg/nsm/refresh).
func newClient(ctx context.Context, name string, nsmAPIClient *nsm.APIClient, additionalFunctionality ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	additionalFunctionality = append(additionalFunctionality,
		sendfd.NewClient(),
		dnscontext.NewClient(dnscontext.WithChainContext(ctx)))

	/* return chain.NewNetworkServiceClient(
		append(
			[]networkservice.NetworkServiceClient{
				updatepath.NewClient(name),
				begin.NewClient(),
				metadata.NewClient(),
				refresh.NewClient(ctx),
				clienturl.NewClient(&nsmAPIClient.Config.ConnectTo),
				//clientconn.NewClient(nsmAPIClient.GRPCClient),
				heal.NewClient(ctx),
				dial.NewClient(ctx,
					dial.WithDialOptions(nsmAPIClient.GRPCDialOption...),
					dial.WithDialTimeout(nsmAPIClient.Config.DialTimeout),
				),
			},
			append(
				additionalFunctionality,
				authorize.NewClient(),
				trimpath.NewClient(),
				connect.NewClient(),
			)...,
		)...,
	) */

	return client.NewClient(ctx,
		client.WithClientURL(&nsmAPIClient.Config.ConnectTo),
		client.WithName(name),
		client.WithAuthorizeClient(authorize.NewClient()),
		client.WithHealClient(heal.NewClient(ctx)),
		client.WithAdditionalFunctionality(additionalFunctionality...),
		client.WithDialTimeout(nsmAPIClient.Config.DialTimeout),
		client.WithDialOptions(nsmAPIClient.GRPCDialOption...),
	)
}

func copyRequest(request *networkservice.NetworkServiceRequest) *networkservice.NetworkServiceRequest {
	if request == nil {
		return nil
	}

	newRequest := &networkservice.NetworkServiceRequest{}

	conn := request.GetConnection()
	if conn != nil {
		newRequest.Connection = &networkservice.Connection{
			Id:                         conn.Id,
			NetworkService:             conn.NetworkService,
			Labels:                     map[string]string{},
			NetworkServiceEndpointName: conn.NetworkServiceEndpointName,
			Payload:                    conn.Payload,
			Context: &networkservice.ConnectionContext{
				IpContext:       &networkservice.IPContext{},
				DnsContext:      &networkservice.DNSContext{},
				EthernetContext: &networkservice.EthernetContext{},
				ExtraContext:    map[string]string{},
			},
		}
		// copy Labels
		for key, value := range conn.Labels {
			newRequest.Connection.Labels[key] = value
		}
		if conn.GetContext() != nil {
			// copy ExtraContext
			for key, value := range conn.GetContext().ExtraContext {
				newRequest.Connection.Context.ExtraContext[key] = value
			}
		}
	}

	mechanismPreferences := request.GetMechanismPreferences()
	if mechanismPreferences != nil {
		newRequest.MechanismPreferences = []*networkservice.Mechanism{}
		for _, value := range mechanismPreferences {
			newMechanismPreference := &networkservice.Mechanism{
				Cls:        value.Cls,
				Type:       value.Type,
				Parameters: map[string]string{},
			}
			// copy Parameters
			for key, value := range value.Parameters {
				newMechanismPreference.Parameters[key] = value
			}
			newRequest.MechanismPreferences = append(newRequest.MechanismPreferences, newMechanismPreference)
		}
	}

	return newRequest
}
