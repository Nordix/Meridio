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

package reader

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

const (
	TrenchConfigKey     = "trench"
	ConduitsConfigKey   = "conduits"
	StreamsConfigKey    = "streams"
	FlowsConfigKey      = "flows"
	VipsConfigKey       = "vips"
	AttractorsConfigKey = "attractors"
	GatewaysConfigKey   = "gateways"
)

func UnmarshalConfig(data map[string]string) (
	*Trench,
	[]*Conduit,
	[]*Stream,
	[]*Flow,
	[]*Vip,
	[]*Attractor,
	[]*Gateway,
	error) {
	trench, err := UnmarshalTrench(data[TrenchConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	conduits, err := UnmarshalConduits(data[ConduitsConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	streams, err := UnmarshalStreams(data[StreamsConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	flows, err := UnmarshalFlows(data[FlowsConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	vips, err := UnmarshalVips(data[VipsConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	attractors, err := UnmarshalAttractors(data[AttractorsConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	gateways, err := UnmarshalGateways(data[GatewaysConfigKey])
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	return trench, conduits, streams, flows, vips, attractors, gateways, nil
}

func UnmarshalTrench(c string) (*Trench, error) {
	config := &Trench{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal trench (%s): %w", c, err)
	}
	return config, nil
}

func UnmarshalConduits(c string) ([]*Conduit, error) {
	config := &ConduitList{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config.Conduits, fmt.Errorf("failed to unmarshal conduits (%s): %w", c, err)
	}
	return config.Conduits, nil
}

func UnmarshalStreams(c string) ([]*Stream, error) {
	config := &StreamList{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config.Streams, fmt.Errorf("failed to unmarshal streams (%s): %w", c, err)
	}
	return config.Streams, nil
}

func UnmarshalFlows(c string) ([]*Flow, error) {
	config := &FlowList{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config.Flows, fmt.Errorf("failed to unmarshal flows (%s): %w", c, err)
	}
	return config.Flows, nil
}

func UnmarshalVips(c string) ([]*Vip, error) {
	config := &VipList{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config.Vips, fmt.Errorf("failed to unmarshal vips (%s): %w", c, err)
	}
	return config.Vips, nil
}

func UnmarshalAttractors(c string) ([]*Attractor, error) {
	config := &AttractorList{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config.Attractors, fmt.Errorf("failed to unmarshal attractors (%s): %w", c, err)
	}
	return config.Attractors, nil
}

func UnmarshalGateways(c string) ([]*Gateway, error) {
	config := &GatewayList{}
	err := yaml.UnmarshalStrict([]byte(c), &config)
	if err != nil {
		return config.Gateways, fmt.Errorf("failed to unmarshal gateways (%s): %w", c, err)
	}
	return config.Gateways, nil
}
