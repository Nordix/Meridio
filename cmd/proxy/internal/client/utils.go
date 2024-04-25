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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/nordix/meridio/pkg/nsm"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func expirationTimeIsNull(expirationTime *timestamppb.Timestamp) bool {
	nullTImeStamp := &timestamppb.Timestamp{
		Seconds: -1,
	}
	return expirationTime == nil || expirationTime.AsTime().Equal(nullTImeStamp.AsTime())
}

// newClient -
// Creates networkservice.NetworkServiceClient relying on NSM's client.NewClient API
//
// Note:
// Refresh Client comes from the NSM sdk version used. (In case of NSM v1.1.1 the built-in
// refresh might lead to connection issues if the different path segments have different
// maxTokenLifetime configured (unless the NSC side has the lowest maxtokenlifetime)).
func newClient(ctx context.Context, name string, nsmAPIClient *nsm.APIClient, healOptions []heal.Option, additionalFunctionality ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	additionalFunctionality = append(additionalFunctionality,
		sendfd.NewClient(),
	)

	return client.NewClient(ctx,
		client.WithClientURL(&nsmAPIClient.Config.ConnectTo),
		client.WithName(name),
		client.WithHealClient(heal.NewClient(ctx, healOptions...)),
		client.WithAdditionalFunctionality(additionalFunctionality...),
		client.WithDialTimeout(nsmAPIClient.Config.DialTimeout),
		client.WithDialOptions(nsmAPIClient.GRPCDialOption...),
	)
}
