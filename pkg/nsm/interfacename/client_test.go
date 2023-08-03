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
		interfacename.NewClient("nsm", generator),
	)
	// connection has non nil Mechanism, interfacenameClient must generate a feasible interface name for it
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:        "conn-id",
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
	assert.Equal(t, conn.GetMechanism().GetParameters()[common.InterfaceNameKey], "nsm")
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "")
	assert.Equal(t, request.GetMechanismPreferences()[1].GetParameters()[common.InterfaceNameKey], "")
}

func Test_Client_Request_Nil_Mechanism(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("nsm", generator),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-id",
		},
	}

	// Note: there's no connection mechanism or preferred mechanism, thus
	// interfacenameClient won't do anything
	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, conn.GetMechanism())
}

func Test_Client_Request_Overwrite(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("nsm", generator),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-id",
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
	assert.Equal(t, conn.GetMechanism().GetParameters()[common.InterfaceNameKey], "nsm")
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "default-A")
	assert.Equal(t, request.GetMechanismPreferences()[1].GetParameters()[common.InterfaceNameKey], "default-B")
}

func Test_Client_Request_Preferred_Mechanism(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("nsm", generator),
	)

	// let interfacenameClient generate a feasible interface name for a new connection request,
	// and save it into MechanismPreferences
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-id",
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Parameters: map[string]string{common.InterfaceNameKey: "random"},
				Cls:        "dummy-cls",
				Type:       "dummy-type",
			},
		},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, conn.GetMechanism())
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "nsm")

	// let interfacenameClient take interfaces in MechanismPreferences into consideration
	// for a connection with Mechanism but no interface name
	request = &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-id",
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{},
				Cls:        "dummy-cls",
				Type:       "dummy-type",
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Parameters: map[string]string{common.InterfaceNameKey: "NewInterfaceName-A"},
			},
			{
				Parameters: map[string]string{common.InterfaceNameKey: "nsm"},
				Cls:        "dummy-cls",
				Type:       "dummy-type",
			},
		},
	}

	conn, err = networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.GetMechanism())
	assert.NotNil(t, conn.GetMechanism().GetParameters())
	assert.Contains(t, conn.GetMechanism().GetParameters(), common.InterfaceNameKey)
	assert.Equal(t, conn.GetMechanism().GetParameters()[common.InterfaceNameKey], "nsm")
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "NewInterfaceName-A")
	assert.Equal(t, request.GetMechanismPreferences()[1].GetParameters()[common.InterfaceNameKey], "nsm")
}

func Test_Client_Request_Nil_Mechanism_Overwrite_Preferred_Mechanism(t *testing.T) {
	generator := &mockGenerator{}
	networkServiceClient := chain.NewNetworkServiceClient(
		interfacename.NewClient("nsm", generator),
	)
	// no connection yet, and MechanismPreferences contains interface name which does not match the prefix
	// interfacenameClient must generate new name and overwrite MechanismPreferences
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-id",
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Parameters: map[string]string{common.InterfaceNameKey: "default-A"},
			},
		},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, conn.GetMechanism())
	assert.Equal(t, request.GetMechanismPreferences()[0].GetParameters()[common.InterfaceNameKey], "nsm")
}
