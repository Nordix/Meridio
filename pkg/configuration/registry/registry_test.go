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

package registry_test

import (
	"testing"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/configuration/registry"
	"github.com/stretchr/testify/assert"
)

type testTrench struct {
	registeredTrench *nspAPI.Trench
	parameter        *nspAPI.Trench
	result           *nspAPI.Trench
}

type testConduit struct {
	registeredConduits []*nspAPI.Conduit
	parameter          *nspAPI.Conduit
	result             []*nspAPI.Conduit
}

type testStream struct {
	registeredStreams []*nspAPI.Stream
	parameter         *nspAPI.Stream
	result            []*nspAPI.Stream
}

type testFlow struct {
	registeredFlows []*nspAPI.Flow
	parameter       *nspAPI.Flow
	result          []*nspAPI.Flow
}

type testVip struct {
	registeredVips []*nspAPI.Vip
	parameter      *nspAPI.Vip
	result         []*nspAPI.Vip
}

type testAttractor struct {
	registeredAttractors []*nspAPI.Attractor
	parameter            *nspAPI.Attractor
	result               []*nspAPI.Attractor
}

type testGateway struct {
	registeredGateways []*nspAPI.Gateway
	parameter          *nspAPI.Gateway
	result             []*nspAPI.Gateway
}

func Test_SetTrench(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetTrench(&nspAPI.Trench{
		Name: "trench-a",
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Trench)
}

func Test_GetTrench(t *testing.T) {
	tests := []testTrench{
		{
			registeredTrench: &nspAPI.Trench{
				Name: "trench-a",
			},
			parameter: &nspAPI.Trench{
				Name: "trench-a",
			},
			result: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			registeredTrench: &nspAPI.Trench{
				Name: "trench-a",
			},
			parameter: nil,
			result: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetTrench(test.registeredTrench)
		result := configurationRegistry.GetTrench(test.parameter)
		assert.Equal(t, result, test.result)
	}
}

func Test_SetConduits(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetConduits([]*nspAPI.Conduit{
		{
			Name: "conduit-a",
		},
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Conduit)
}

func Test_GetConduits(t *testing.T) {
	registeredConduits := []*nspAPI.Conduit{
		{
			Name: "conduit-a",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name: "conduit-b",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name: "conduit-c",
			Trench: &nspAPI.Trench{
				Name: "trench-b",
			},
		},
	}
	tests := []testConduit{
		{
			registeredConduits: registeredConduits,
			parameter: &nspAPI.Conduit{
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Conduit{registeredConduits[0], registeredConduits[1]},
		},
		{
			registeredConduits: registeredConduits,
			parameter: &nspAPI.Conduit{
				Name: "conduit-b",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Conduit{registeredConduits[1]},
		},
		{
			registeredConduits: registeredConduits,
			parameter:          nil,
			result:             registeredConduits,
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetConduits(test.registeredConduits)
		results := configurationRegistry.GetConduits(test.parameter)
		assert.Len(t, results, len(test.result))
		for _, r := range results {
			assert.Contains(t, test.result, r)
		}
	}
}

func Test_SetStreams(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetStreams([]*nspAPI.Stream{
		{
			Name: "stream-a",
		},
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Stream)
}

func Test_GetStreams(t *testing.T) {
	registeredStreams := []*nspAPI.Stream{
		{
			Name: "stream-a",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
		{
			Name: "stream-b",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-a",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
		{
			Name: "stream-c",
			Conduit: &nspAPI.Conduit{
				Name: "conduit-b",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
		},
	}
	tests := []testStream{
		{
			registeredStreams: registeredStreams,
			parameter: &nspAPI.Stream{
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
			result: []*nspAPI.Stream{registeredStreams[0], registeredStreams[1]},
		},
		{
			registeredStreams: registeredStreams,
			parameter: &nspAPI.Stream{
				Name: "stream-b",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
			result: []*nspAPI.Stream{registeredStreams[1]},
		},
		{
			registeredStreams: registeredStreams,
			parameter: &nspAPI.Stream{
				Conduit: &nspAPI.Conduit{
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
			result: registeredStreams,
		},
		{
			registeredStreams: registeredStreams,
			parameter:         nil,
			result:            registeredStreams,
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetStreams(test.registeredStreams)
		results := configurationRegistry.GetStreams(test.parameter)
		assert.Len(t, results, len(test.result))
		for _, r := range results {
			assert.Contains(t, test.result, r)
		}
	}
}

func Test_SetFlows(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetFlows([]*nspAPI.Flow{
		{
			Name: "flow-a",
		},
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Flow)
}

func Test_GetFlows(t *testing.T) {
	registeredFlows := []*nspAPI.Flow{
		{
			Name:                  "flow-a",
			SourceSubnets:         []string{"224.0.0.0/16", "1500::/16"},
			DestinationPortRanges: []string{"85"},
			SourcePortRanges:      []string{"45000"},
			Protocols:             []string{"tcp"},
			Vips: []*nspAPI.Vip{
				{
					Name:    "vip-1",
					Address: "2001::1/128",
				},
			},
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Name:                  "flow-b",
			SourceSubnets:         []string{"124.0.0.0/24", "2001::/32"},
			DestinationPortRanges: []string{"80", "90-95"},
			SourcePortRanges:      []string{"35000-35500", "40000"},
			Protocols:             []string{"tcp", "udp"},
			Vips: []*nspAPI.Vip{
				{
					Name:    "vip-2",
					Address: "20.0.0.1/32",
				},
			},
			Stream: &nspAPI.Stream{
				Name: "stream-a",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
		{
			Name:                  "flow-c",
			SourceSubnets:         []string{"125.0.0.0/24", "2002::/32"},
			DestinationPortRanges: []string{"150-170"},
			SourcePortRanges:      []string{"61000"},
			Protocols:             []string{"udp"},
			Vips: []*nspAPI.Vip{
				{
					Name:    "vip-3",
					Address: "40.0.0.0/24",
				},
			},
			Stream: &nspAPI.Stream{
				Name: "stream-b",
				Conduit: &nspAPI.Conduit{
					Name: "conduit-a",
					Trench: &nspAPI.Trench{
						Name: "trench-a",
					},
				},
			},
		},
	}
	tests := []testFlow{
		{
			registeredFlows: registeredFlows,
			parameter: &nspAPI.Flow{
				Stream: &nspAPI.Stream{
					Name: "stream-a",
					Conduit: &nspAPI.Conduit{
						Name: "conduit-a",
						Trench: &nspAPI.Trench{
							Name: "trench-a",
						},
					},
				},
			},
			result: []*nspAPI.Flow{registeredFlows[0], registeredFlows[1]},
		},
		{
			registeredFlows: registeredFlows,
			parameter: &nspAPI.Flow{
				Name: "flow-b",
				Stream: &nspAPI.Stream{
					Name: "stream-a",
					Conduit: &nspAPI.Conduit{
						Name: "conduit-a",
						Trench: &nspAPI.Trench{
							Name: "trench-a",
						},
					},
				},
			},
			result: []*nspAPI.Flow{registeredFlows[1]},
		},
		{
			registeredFlows: registeredFlows,
			parameter: &nspAPI.Flow{
				Stream: &nspAPI.Stream{
					Conduit: &nspAPI.Conduit{
						Name: "conduit-a",
						Trench: &nspAPI.Trench{
							Name: "trench-a",
						},
					},
				},
			},
			result: registeredFlows,
		},
		{
			registeredFlows: registeredFlows,
			parameter:       nil,
			result:          registeredFlows,
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetFlows(test.registeredFlows)
		results := configurationRegistry.GetFlows(test.parameter)
		assert.Len(t, results, len(test.result))
		for _, r := range results {
			assert.Contains(t, test.result, r)
		}
	}
}

func Test_SetVips(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetVips([]*nspAPI.Vip{
		{
			Name: "vip-a",
		},
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Vip)
}

func Test_GetVips(t *testing.T) {
	registeredVips := []*nspAPI.Vip{
		{
			Name:    "vip-a",
			Address: "2001::1/128",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name:    "vip-b",
			Address: "20.0.0.1/32",
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name:    "vip-c",
			Address: "40.0.0.0/24",
			Trench: &nspAPI.Trench{
				Name: "trench-b",
			},
		},
	}
	tests := []testVip{
		{
			registeredVips: registeredVips,
			parameter: &nspAPI.Vip{
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Vip{registeredVips[0], registeredVips[1]},
		},
		{
			registeredVips: registeredVips,
			parameter: &nspAPI.Vip{
				Name: "vip-b",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Vip{registeredVips[1]},
		},
		{
			registeredVips: registeredVips,
			parameter:      nil,
			result:         registeredVips,
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetVips(test.registeredVips)
		results := configurationRegistry.GetVips(test.parameter)
		assert.Len(t, results, len(test.result))
		for _, r := range results {
			assert.Contains(t, test.result, r)
		}
	}
}

func Test_SetAttractors(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetAttractors([]*nspAPI.Attractor{
		{
			Name: "attractor-a",
		},
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Attractor)
}

func Test_GetAttractors(t *testing.T) {
	registeredAttractors := []*nspAPI.Attractor{
		{
			Name: "attractor-a",
			Vips: []*nspAPI.Vip{
				{
					Name:    "vip-1",
					Address: "2001::1/128",
				},
			},
			Gateways: []*nspAPI.Gateway{
				{
					Name: "gateway-1",
				},
			},
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name: "attractor-b",
			Vips: []*nspAPI.Vip{
				{
					Name:    "vip-2",
					Address: "20.0.0.1/32",
				},
			},
			Gateways: []*nspAPI.Gateway{
				{
					Name: "gateway-2",
				},
			},
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name: "attractor-c",
			Vips: []*nspAPI.Vip{
				{
					Name:    "vip-3",
					Address: "40.0.0.0/24",
				},
			},
			Gateways: []*nspAPI.Gateway{
				{
					Name: "gateway-3",
				},
			},
			Trench: &nspAPI.Trench{
				Name: "trench-b",
			},
		},
	}
	tests := []testAttractor{
		{
			registeredAttractors: registeredAttractors,
			parameter: &nspAPI.Attractor{
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Attractor{registeredAttractors[0], registeredAttractors[1]},
		},
		{
			registeredAttractors: registeredAttractors,
			parameter: &nspAPI.Attractor{
				Name: "attractor-b",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Attractor{registeredAttractors[1]},
		},
		{
			registeredAttractors: registeredAttractors,
			parameter:            nil,
			result:               registeredAttractors,
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetAttractors(test.registeredAttractors)
		results := configurationRegistry.GetAttractors(test.parameter)
		assert.Len(t, results, len(test.result))
		for _, r := range results {
			assert.Contains(t, test.result, r)
		}
	}
}

func Test_SetGateways(t *testing.T) {
	configurationEventChan := make(chan *registry.ConfigurationEvent, 10)
	configurationRegistry := registry.New(configurationEventChan)
	configurationRegistry.SetGateways([]*nspAPI.Gateway{
		{
			Name: "gateway-a",
		},
	})
	var configurationEvent *registry.ConfigurationEvent
	select {
	case configurationEvent = <-configurationEventChan:
	default:
	}
	assert.NotNil(t, configurationEvent)
	assert.Equal(t, int(configurationEvent.ResourceType), registry.Gateway)
}

func Test_GetGateways(t *testing.T) {
	registeredGateways := []*nspAPI.Gateway{
		{
			Name:       "gateway-a",
			Address:    "169.254.100.150",
			RemoteASN:  4248829953,
			LocalASN:   8103,
			RemotePort: 10179,
			LocalPort:  10179,
			IpFamily:   "ipv4",
			Bfd:        false,
			Protocol:   "bgp",
			HoldTime:   3,
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name:       "gateway-b",
			Address:    "100:100::150",
			RemoteASN:  4248829953,
			LocalASN:   8103,
			RemotePort: 10179,
			LocalPort:  10179,
			IpFamily:   "ipv6",
			Bfd:        false,
			Protocol:   "bgp",
			HoldTime:   5,
			Trench: &nspAPI.Trench{
				Name: "trench-a",
			},
		},
		{
			Name:       "gateway-c",
			Address:    "170.250.0.100",
			RemoteASN:  4248829953,
			LocalASN:   8200,
			RemotePort: 12000,
			LocalPort:  12000,
			IpFamily:   "ipv4",
			Bfd:        false,
			Protocol:   "bgp",
			HoldTime:   3,
			Trench: &nspAPI.Trench{
				Name: "trench-b",
			},
		},
	}
	tests := []testGateway{
		{
			registeredGateways: registeredGateways,
			parameter: &nspAPI.Gateway{
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Gateway{registeredGateways[0], registeredGateways[1]},
		},
		{
			registeredGateways: registeredGateways,
			parameter: &nspAPI.Gateway{
				Name: "gateway-b",
				Trench: &nspAPI.Trench{
					Name: "trench-a",
				},
			},
			result: []*nspAPI.Gateway{registeredGateways[1]},
		},
		{
			registeredGateways: registeredGateways,
			parameter:          nil,
			result:             registeredGateways,
		},
	}
	configurationRegistry := registry.New(nil)
	for _, test := range tests {
		configurationRegistry.SetGateways(test.registeredGateways)
		results := configurationRegistry.GetGateways(test.parameter)
		assert.Len(t, results, len(test.result))
		for _, r := range results {
			assert.Contains(t, test.result, r)
		}
	}
}
