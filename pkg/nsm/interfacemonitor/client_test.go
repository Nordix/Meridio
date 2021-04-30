package interfacemonitor_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/stretchr/testify/assert"
)

func Test_Client_Request_NonExistingInterface(t *testing.T) {
	interfaceName := "default"
	sm := &subscriberMock{}
	imm := &interfaceMonitorMock{}
	num := &networkingUtilsMock{
		interfaceExists: false,
		interfaceName:   interfaceName,
	}

	interfaceMonitorClient := interfacemonitor.NewClient(imm, sm, num)

	networkServiceClient := chain.NewNetworkServiceClient(
		interfaceMonitorClient,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddr:     "172.16.0.2",
					DstIpAddr:     "172.16.0.1",
					ExtraPrefixes: []string{"172.16.0.1"},
				},
			},
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{common.InterfaceNameKey: interfaceName},
			},
		},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, sm.interfaceReceivedCreation)
	num.interfaceExists = true
	imm.advertiseInterfaceCreated(num.NewInterface(1))
	assert.NotNil(t, sm.interfaceReceivedCreation)
	assert.Equal(t, interfaceName, sm.interfaceReceivedCreation.GetName())
	assert.Equal(t, []string{"172.16.0.2"}, sm.interfaceReceivedCreation.GetLocalPrefixes())
	assert.Equal(t, []string{"172.16.0.1"}, sm.interfaceReceivedCreation.GetNeighborPrefixes())
	assert.Equal(t, []string{"172.16.0.1"}, sm.interfaceReceivedCreation.GetGatewayPrefixes())
}

func Test_Client_Request_ExistingInterface(t *testing.T) {
	interfaceName := "default"
	sm := &subscriberMock{}
	imm := &interfaceMonitorMock{}
	num := &networkingUtilsMock{
		interfaceExists: true,
		interfaceName:   interfaceName,
	}

	interfaceMonitorClient := interfacemonitor.NewClient(imm, sm, num)

	networkServiceClient := chain.NewNetworkServiceClient(
		interfaceMonitorClient,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddr:     "172.16.0.2",
					DstIpAddr:     "172.16.0.1",
					ExtraPrefixes: []string{"172.16.0.1"},
				},
			},
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{common.InterfaceNameKey: interfaceName},
			},
		},
	}

	conn, err := networkServiceClient.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, sm.interfaceReceivedCreation)
	assert.Equal(t, interfaceName, sm.interfaceReceivedCreation.GetName())
	assert.Equal(t, []string{"172.16.0.2"}, sm.interfaceReceivedCreation.GetLocalPrefixes())
	assert.Equal(t, []string{"172.16.0.1"}, sm.interfaceReceivedCreation.GetNeighborPrefixes())
	assert.Equal(t, []string{"172.16.0.1"}, sm.interfaceReceivedCreation.GetGatewayPrefixes())
}

func Test_Client_Close(t *testing.T) {
	interfaceName := "default"
	sm := &subscriberMock{}
	imm := &interfaceMonitorMock{}
	num := &networkingUtilsMock{
		interfaceExists: true,
		interfaceName:   interfaceName,
	}

	interfaceMonitorClient := interfacemonitor.NewClient(imm, sm, num)

	networkServiceClient := chain.NewNetworkServiceClient(
		interfaceMonitorClient,
	)

	conn := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Parameters: map[string]string{common.InterfaceNameKey: interfaceName},
		},
	}

	_, err := networkServiceClient.Close(context.Background(), conn)
	assert.Nil(t, err)
	assert.NotNil(t, sm.interfaceReceivedDeletion)
	assert.Equal(t, interfaceName, sm.interfaceReceivedDeletion.GetName())
}