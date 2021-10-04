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

package reader_test

import (
	"testing"

	"github.com/nordix/meridio/pkg/configuration/reader"
	"github.com/stretchr/testify/assert"
)

type test struct {
	yaml          string
	valueExpected interface{}
	errExpected   bool
}

func Test_UnmarshalTrench(t *testing.T) {
	tests := []test{
		{
			yaml: `
name: "trench-a"`,
			valueExpected: &reader.Trench{
				Name: "trench-a",
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		trench, err := reader.UnmarshalTrench(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, trench, test.valueExpected)
	}
}

func Test_UnmarshalConduits(t *testing.T) {
	tests := []test{
		{
			yaml: `
items:
- name: "conduit-a"
  trench: "trench-a"`,
			valueExpected: []*reader.Conduit{
				{
					Name:   "conduit-a",
					Trench: "trench-a",
				},
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		conduits, err := reader.UnmarshalConduits(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		conduitsExpected := test.valueExpected.([]*reader.Conduit)
		assert.Len(t, conduits, len(conduitsExpected))
		for _, conduit := range conduits {
			assert.Contains(t, conduitsExpected, conduit)
		}
	}
}

func Test_UnmarshalStreams(t *testing.T) {
	tests := []test{
		{
			yaml: `
items:
- name: "stream-a"
  conduit: "conduit-a"`,
			valueExpected: []*reader.Stream{
				{
					Name:    "stream-a",
					Conduit: "conduit-a",
				},
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		streams, err := reader.UnmarshalStreams(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		streamsExpected := test.valueExpected.([]*reader.Stream)
		assert.Len(t, streams, len(streamsExpected))
		for _, stream := range streams {
			assert.Contains(t, streamsExpected, stream)
		}
	}
}

func Test_UnmarshalFlows(t *testing.T) {
	tests := []test{
		{
			yaml: `
items:
- name: "flow-a"
  source-subnets: ["124.0.0.0/24", "2001::/32"]
  destination-port-ranges: ["80", "90-95"]
  source-port-ranges: ["35000-35500", "40000"]
  protocols: ["tcp", "udp"]
  vips: ["vip-1"]
  stream: "stream-a"`,
			valueExpected: []*reader.Flow{
				{
					Name:                  "flow-a",
					SourceSubnets:         []string{"124.0.0.0/24", "2001::/32"},
					DestinationPortRanges: []string{"80", "90-95"},
					SourcePortRanges:      []string{"35000-35500", "40000"},
					Protocols:             []string{"tcp", "udp"},
					Vips:                  []string{"vip-1"},
					Stream:                "stream-a",
				},
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		flows, err := reader.UnmarshalFlows(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		flowsExpected := test.valueExpected.([]*reader.Flow)
		assert.Len(t, flows, len(flowsExpected))
		for _, flow := range flows {
			assert.Contains(t, flowsExpected, flow)
		}
	}
}

func Test_UnmarshalVips(t *testing.T) {
	tests := []test{
		{
			yaml: `
items:
- name: "vip-a"
  address: "20.0.0.1/32"
  trench: "trench-a"`,
			valueExpected: []*reader.Vip{
				{
					Name:    "vip-a",
					Address: "20.0.0.1/32",
					Trench:  "trench-a",
				},
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		vips, err := reader.UnmarshalVips(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		vipsExpected := test.valueExpected.([]*reader.Vip)
		assert.Len(t, vips, len(vipsExpected))
		for _, vip := range vips {
			assert.Contains(t, vipsExpected, vip)
		}
	}
}

func Test_UnmarshalAttractors(t *testing.T) {
	tests := []test{
		{
			yaml: `
items:
- name: "attractor-a"
  gateways: ["gateways-1"]
  vips: ["vip-1"]
  trench: "trench-a"`,
			valueExpected: []*reader.Attractor{
				{
					Name:     "attractor-a",
					Gateways: []string{"gateways-1"},
					Vips:     []string{"vip-1"},
					Trench:   "trench-a",
				},
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		attractors, err := reader.UnmarshalAttractors(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		attractorsExpected := test.valueExpected.([]*reader.Attractor)
		assert.Len(t, attractors, len(attractorsExpected))
		for _, attractor := range attractors {
			assert.Contains(t, attractorsExpected, attractor)
		}
	}
}

func Test_UnmarshalGateways(t *testing.T) {
	tests := []test{
		{
			yaml: `
items:
- name: "attractor-a"
  address: "169.254.100.150"
  remote-asn: 4248829953
  local-asn: 8103
  remote-port: 10179
  local-port: 10179
  ip-family: "ipv4"
  bfd: false
  protocol: "bgp"
  hold-time: 3
  trench: "trench-a"`,
			valueExpected: []*reader.Gateway{
				{
					Name:       "attractor-a",
					Address:    "169.254.100.150",
					RemoteASN:  4248829953,
					LocalASN:   8103,
					RemotePort: 10179,
					LocalPort:  10179,
					IPFamily:   "ipv4",
					BFD:        false,
					Protocol:   "bgp",
					HoldTime:   3,
					Trench:     "trench-a",
				},
			},
			errExpected: false,
		},
	}
	for _, test := range tests {
		gateways, err := reader.UnmarshalGateways(test.yaml)
		if test.errExpected {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		gatewaysExpected := test.valueExpected.([]*reader.Gateway)
		assert.Len(t, gateways, len(gatewaysExpected))
		for _, gateway := range gateways {
			assert.Contains(t, gatewaysExpected, gateway)
		}
	}
}
