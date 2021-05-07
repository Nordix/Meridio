package interfacemonitor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/nordix/meridio/pkg/kernel"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm/interfacemonitor"
	"github.com/stretchr/testify/assert"
)

type subscriberMock struct {
	interfaceReceivedCreation networking.Iface
	interfaceReceivedDeletion networking.Iface
}

func (sm *subscriberMock) InterfaceCreated(intf networking.Iface) {
	sm.interfaceReceivedCreation = intf
}

func (sm *subscriberMock) InterfaceDeleted(intf networking.Iface) {
	sm.interfaceReceivedDeletion = intf
}

type interfaceMonitorMock struct {
	subscriber networking.InterfaceMonitorSubscriber
}

func (imm *interfaceMonitorMock) advertiseInterfaceCreated(intf networking.Iface) {
	imm.subscriber.InterfaceCreated(intf)
}

func (imm *interfaceMonitorMock) Subscribe(subscriber networking.InterfaceMonitorSubscriber) {
	imm.subscriber = subscriber
}

func (imm *interfaceMonitorMock) UnSubscribe(subscriber networking.InterfaceMonitorSubscriber) {
}

func (imm *interfaceMonitorMock) Close() {
}

type networkingUtilsMock struct {
	interfaceExists bool
	interfaceName   string
}

func (num *networkingUtilsMock) NewInterface(index int) networking.Iface {
	return &ifaceMock{
		Interface: &kernel.Interface{},
		name:      num.interfaceName,
	}
}

func (num *networkingUtilsMock) GetIndexFromName(name string) (int, error) {
	if num.interfaceExists {
		return 1, nil
	}
	return -1, errors.New("interface does not exist")
}

type ifaceMock struct {
	*kernel.Interface
	name string
}

func (im *ifaceMock) GetName() string {
	return im.name
}

func Test_Server_Request_NonExistingInterface(t *testing.T) {
	interfaceName := "default"
	sm := &subscriberMock{}
	imm := &interfaceMonitorMock{}
	num := &networkingUtilsMock{
		interfaceExists: false,
		interfaceName:   interfaceName,
	}

	interfaceMonitorEndpoint := interfacemonitor.NewServer(imm, sm, num)

	networkServiceServer := chain.NewNetworkServiceServer(
		interfaceMonitorEndpoint,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs:    []string{"172.16.0.2"},
					DstIpAddrs:    []string{"172.16.0.1"},
					ExtraPrefixes: []string{"172.16.0.2"},
				},
			},
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{common.InterfaceNameKey: interfaceName},
			},
		},
	}

	conn, err := networkServiceServer.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Nil(t, sm.interfaceReceivedCreation)
	num.interfaceExists = true
	imm.advertiseInterfaceCreated(num.NewInterface(1))
	assert.NotNil(t, sm.interfaceReceivedCreation)
	assert.Equal(t, interfaceName, sm.interfaceReceivedCreation.GetName())
	assert.Equal(t, []string{"172.16.0.1"}, sm.interfaceReceivedCreation.GetLocalPrefixes())
	assert.Equal(t, []string{"172.16.0.2"}, sm.interfaceReceivedCreation.GetNeighborPrefixes())
	assert.Equal(t, []string{"172.16.0.2"}, sm.interfaceReceivedCreation.GetGatewayPrefixes())
}

func Test_Server_Request_ExistingInterface(t *testing.T) {
	interfaceName := "default"
	sm := &subscriberMock{}
	imm := &interfaceMonitorMock{}
	num := &networkingUtilsMock{
		interfaceExists: true,
		interfaceName:   interfaceName,
	}

	interfaceMonitorEndpoint := interfacemonitor.NewServer(imm, sm, num)

	networkServiceServer := chain.NewNetworkServiceServer(
		interfaceMonitorEndpoint,
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs:    []string{"172.16.0.2"},
					DstIpAddrs:    []string{"172.16.0.1"},
					ExtraPrefixes: []string{"172.16.0.2"},
				},
			},
			Mechanism: &networkservice.Mechanism{
				Parameters: map[string]string{common.InterfaceNameKey: interfaceName},
			},
		},
	}

	conn, err := networkServiceServer.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, sm.interfaceReceivedCreation)
	assert.Equal(t, interfaceName, sm.interfaceReceivedCreation.GetName())
	assert.Equal(t, []string{"172.16.0.1"}, sm.interfaceReceivedCreation.GetLocalPrefixes())
	assert.Equal(t, []string{"172.16.0.2"}, sm.interfaceReceivedCreation.GetNeighborPrefixes())
	assert.Equal(t, []string{"172.16.0.2"}, sm.interfaceReceivedCreation.GetGatewayPrefixes())
}

func Test_Server_Close(t *testing.T) {
	interfaceName := "default"
	sm := &subscriberMock{}
	imm := &interfaceMonitorMock{}
	num := &networkingUtilsMock{
		interfaceExists: true,
		interfaceName:   interfaceName,
	}

	interfaceMonitorEndpoint := interfacemonitor.NewServer(imm, sm, num)

	networkServiceServer := chain.NewNetworkServiceServer(
		interfaceMonitorEndpoint,
	)

	conn := &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Parameters: map[string]string{common.InterfaceNameKey: interfaceName},
		},
	}

	_, err := networkServiceServer.Close(context.Background(), conn)
	assert.Nil(t, err)
	assert.NotNil(t, sm.interfaceReceivedDeletion)
	assert.Equal(t, interfaceName, sm.interfaceReceivedDeletion.GetName())
}
