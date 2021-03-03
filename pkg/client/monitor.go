package client

import (
	"io"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/sirupsen/logrus"
)

type Monitor struct {
	networkServiceName            string
	networkServiceClients         map[string]*NetworkServiceClient
	NetworkServiceDiscoveryStream registry.NetworkServiceEndpointRegistry_FindClient
	registryClient                RegistryClient
	nsmgrClient                   NSMgrClient
	interfaceMonitorSubscriber    networking.InterfaceMonitorSubscriber
	nscConnectionFactory          NSCConnectionFactory
}

type RegistryClient interface {
	Find(*registry.NetworkServiceEndpointQuery) (registry.NetworkServiceEndpointRegistry_FindClient, error)
}

// Start the monitoring
func (m *Monitor) Start() {
	logrus.Infof("Full Mesh Client: Start monitoring Network Service: %v", m.networkServiceName)

	query := m.prepareQuery()
	var err error
	m.NetworkServiceDiscoveryStream, err = m.registryClient.Find(query)
	if err != nil {
		logrus.Errorf("Full Mesh Endpoint Monitor (%v): err Find: %v", m.networkServiceName, err)
	}

	m.recv()
}

// Stop the monitoring
func (m *Monitor) Stop() {
	// TODO
}

func (m *Monitor) recv() {
	for {
		networkServiceEndpoint, err := m.NetworkServiceDiscoveryStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("Full Mesh Client Monitor (%v): event err: %v", m.networkServiceName, err)
			break
		}

		if m.networkServiceClientExists(networkServiceEndpoint.Name) == false {
			m.endpointAdded(networkServiceEndpoint)
		} else if m.expirationTimeIsNull(networkServiceEndpoint.ExpirationTime) {
			m.endpointRemoved(networkServiceEndpoint)
		}
	}
}

func (m *Monitor) expirationTimeIsNull(expirationTime *timestamp.Timestamp) bool {
	nullTImeStamp := &timestamp.Timestamp{
		Seconds: -1,
	}
	return expirationTime == nil || expirationTime.AsTime().Equal(nullTImeStamp.AsTime())
}

func (m *Monitor) endpointAdded(networkServiceEndpoint *registry.NetworkServiceEndpoint) {
	logrus.Infof("Full Mesh Client Monitor (%v): event add: %v", m.networkServiceName, networkServiceEndpoint.Name)
	networkServiceClient := NewNetworkServiceClient(m.networkServiceName, m.nsmgrClient)
	networkServiceClient.NetworkServiceEndpointName = networkServiceEndpoint.Name
	networkServiceClient.InterfaceMonitorSubscriber = m.interfaceMonitorSubscriber
	networkServiceClient.nscConnectionFactory = m.nscConnectionFactory
	go networkServiceClient.Request()
	m.networkServiceClients[networkServiceEndpoint.Name] = networkServiceClient
}

func (m *Monitor) endpointRemoved(networkServiceEndpoint *registry.NetworkServiceEndpoint) {
	logrus.Infof("Full Mesh Client Monitor (%v): event delete: %v", m.networkServiceName, networkServiceEndpoint.Name)
	networkServiceClient, exists := m.networkServiceClients[networkServiceEndpoint.Name]
	if exists == false {
		return
	}
	go networkServiceClient.Close()
	delete(m.networkServiceClients, networkServiceEndpoint.Name)
}

func (m *Monitor) networkServiceClientExists(networkServiceEndpointName string) bool {
	_, ok := m.networkServiceClients[networkServiceEndpointName]
	return ok
}

func (m *Monitor) prepareQuery() *registry.NetworkServiceEndpointQuery {
	networkServiceEndpoint := &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{m.networkServiceName},
	}
	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: networkServiceEndpoint,
		Watch:                  true,
	}
	return query
}

func (m *Monitor) SetInterfaceMonitorSubscriber(interfaceMonitorSubscriber networking.InterfaceMonitorSubscriber) {
	m.interfaceMonitorSubscriber = interfaceMonitorSubscriber
	for _, nsc := range m.networkServiceClients {
		nsc.InterfaceMonitorSubscriber = interfaceMonitorSubscriber
	}
}

func (m *Monitor) SetNSCConnectionFactory(nscConnectionFactory NSCConnectionFactory) {
	m.nscConnectionFactory = nscConnectionFactory
}

// NewMonitor - Create a struct monitoring NSEs of a Network Service
func NewMonitor(networkServiceName string, registryClient RegistryClient, nsmgrClient NSMgrClient) *Monitor {
	monitor := &Monitor{
		networkServiceName:    networkServiceName,
		networkServiceClients: make(map[string]*NetworkServiceClient),
		registryClient:        registryClient,
		nsmgrClient:           nsmgrClient,
	}

	return monitor
}
