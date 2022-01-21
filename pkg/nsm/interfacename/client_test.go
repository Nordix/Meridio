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

package interfacename_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/stretchr/testify/assert"
)

func Test_Client_Request(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("NewInterfaceName", generator),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{},
			{},
		},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.GetMechanism())
	assert.NotNil(t, conn.GetMechanism().GetParameters())
	assert.Contains(t, conn.GetMechanism().GetParameters(), common.InterfaceNameKey)
	assert.Equal(t, conn.GetMechanism().GetParameters()[common.InterfaceNameKey], "NewInterfaceName")
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "NewInterfaceName")
	assert.Equal(t, request.GetMechanismPreferences()[1].GetParameters()[common.InterfaceNameKey], "NewInterfaceName")
}

func Test_Client_Request_Nil_Mechanism(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("NewInterfaceName", generator),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, conn.GetMechanism())
}

func Test_Client_Request_Overwrite(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("NewInterfaceName", generator),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{common.InterfaceNameKey: "default"},
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Parameters: map[string]string{common.InterfaceNameKey: "default-A"},
			},
			{
				Parameters: map[string]string{common.InterfaceNameKey: "default-B"},
			},
		},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.GetMechanism())
	assert.NotNil(t, conn.GetMechanism().GetParameters())
	assert.Contains(t, conn.GetMechanism().GetParameters(), common.InterfaceNameKey)
	assert.Equal(t, conn.GetMechanism().GetParameters()[common.InterfaceNameKey], "NewInterfaceName")
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "NewInterfaceName")
	assert.Equal(t, request.GetMechanismPreferences()[1].GetParameters()[common.InterfaceNameKey], "NewInterfaceName")
}
