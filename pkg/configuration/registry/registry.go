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

package registry

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

// types of resource
const (
	Trench = iota
	Conduit
	Stream
	Flow
	Vip
	Attractor
	Gateway
)

type ResourceType int

// ConfigurationEvent reprensent the struct which will be
// sent on any resource modification
type ConfigurationEvent struct {
	ResourceType ResourceType
}

// ConfigurationRegistry is a memory registry for the meridio
// configuration. It can register and get all resources: trench
// conduits, streams, flows, vips, attractors and gateways.
// On any resource modification, a event is sent via a channel
type ConfigurationRegistry struct {
	// Chan is the channel which will send the event on any
	// resource modification
	Chan       chan<- *ConfigurationEvent
	trench     *nspAPI.Trench
	conduits   []*nspAPI.Conduit
	streams    []*nspAPI.Stream
	flows      []*nspAPI.Flow
	vips       []*nspAPI.Vip
	attractors []*nspAPI.Attractor
	gateways   []*nspAPI.Gateway
}

// New is the constructor of ConfigurationRegistry.
func New(configurationEventChan chan<- *ConfigurationEvent) *ConfigurationRegistry {
	configurationRegistry := &ConfigurationRegistry{
		Chan:       configurationEventChan,
		trench:     nil,
		conduits:   []*nspAPI.Conduit{},
		streams:    []*nspAPI.Stream{},
		flows:      []*nspAPI.Flow{},
		vips:       []*nspAPI.Vip{},
		attractors: []*nspAPI.Attractor{},
		gateways:   []*nspAPI.Gateway{},
	}
	return configurationRegistry
}

// SetTrench sets the trench and notifies the change via the channel
// with the resource type as Trench
func (cr *ConfigurationRegistry) SetTrench(trench *nspAPI.Trench) {
	cr.trench = trench
	cr.notify(&ConfigurationEvent{
		ResourceType: Trench,
	})
}

// GetTrench returns a trench with the same name as the one in parameter
// if existing. If the trench in parameter is nil, the registered trench
// will be returned
func (cr *ConfigurationRegistry) GetTrench(trench *nspAPI.Trench) *nspAPI.Trench {
	if cr.trench == nil {
		return nil
	}
	if trench == nil {
		return cr.trench
	}
	if trench.Name == cr.trench.Name {
		return cr.trench
	}
	return nil
}

// SetConduits sets the conduits and notifies the change via the channel
// with the resource type as Conduit
func (cr *ConfigurationRegistry) SetConduits(conduits []*nspAPI.Conduit) {
	cr.conduits = conduits
	cr.notify(&ConfigurationEvent{
		ResourceType: Conduit,
	})
}

// GetConduits returns conduits with the same name (ignored if empty) and
// the same parent as the one in parameter. If the conduit in parameter is
// nil, all conduits will be returned
func (cr *ConfigurationRegistry) GetConduits(conduit *nspAPI.Conduit) []*nspAPI.Conduit {
	conduits := []*nspAPI.Conduit{}
	if conduit == nil {
		return cr.conduits
	}
	for _, c := range cr.conduits {
		if !c.Equals(conduit) {
			continue
		}
		conduits = append(conduits, c)
	}
	return conduits
}

// SetStreams sets the streams and notifies the change via the channel
// with the resource type as Stream
func (cr *ConfigurationRegistry) SetStreams(streams []*nspAPI.Stream) {
	cr.streams = streams
	cr.notify(&ConfigurationEvent{
		ResourceType: Stream,
	})
}

// GetStreams returns streams with the same name (ignored if empty) and
// the same parent as the one in parameter. If the stream in parameter is
// nil, all streams will be returned
func (cr *ConfigurationRegistry) GetStreams(stream *nspAPI.Stream) []*nspAPI.Stream {
	streams := []*nspAPI.Stream{}
	if stream == nil {
		return cr.streams
	}
	for _, s := range cr.streams {
		if !s.Equals(stream) {
			continue
		}
		streams = append(streams, s)
	}
	return streams
}

// SetFlows sets the flows and notifies the change via the channel
// with the resource type as Flow
func (cr *ConfigurationRegistry) SetFlows(flows []*nspAPI.Flow) {
	cr.flows = flows
	cr.notify(&ConfigurationEvent{
		ResourceType: Flow,
	})
}

// GetFlows returns flows with the same name (ignored if empty) and
// the same parent as the one in parameter. If the flow in parameter is
// nil, all flows will be returned
func (cr *ConfigurationRegistry) GetFlows(flow *nspAPI.Flow) []*nspAPI.Flow {
	flows := []*nspAPI.Flow{}
	if flow == nil {
		return cr.flows
	}
	for _, f := range cr.flows {
		if !f.Equals(flow) {
			continue
		}
		flows = append(flows, f)
	}
	return flows
}

// SetVips sets the vips and notifies the change via the channel
// with the resource type as Vip
func (cr *ConfigurationRegistry) SetVips(vips []*nspAPI.Vip) {
	cr.vips = vips
	cr.notify(&ConfigurationEvent{
		ResourceType: Vip,
	})
}

// GetVips returns vips with the same name (ignored if empty) and
// the same parent as the one in parameter. If the vip in parameter is
// nil, all vips will be returned
func (cr *ConfigurationRegistry) GetVips(vip *nspAPI.Vip) []*nspAPI.Vip {
	vips := []*nspAPI.Vip{}
	if vip == nil {
		return cr.vips
	}
	for _, v := range cr.vips {
		if !v.Equals(vip) {
			continue
		}
		vips = append(vips, v)
	}
	return vips
}

// SetAttractors sets the attractors and notifies the change via the channel
// with the resource type as Attractor
func (cr *ConfigurationRegistry) SetAttractors(attractors []*nspAPI.Attractor) {
	cr.attractors = attractors
	cr.notify(&ConfigurationEvent{
		ResourceType: Attractor,
	})
}

// GetAttractors returns attractors with the same name (ignored if empty) and
// the same parent as the one in parameter. If the attractor in parameter is
// nil, all attractors will be returned
func (cr *ConfigurationRegistry) GetAttractors(attractor *nspAPI.Attractor) []*nspAPI.Attractor {
	attractors := []*nspAPI.Attractor{}
	if attractor == nil {
		return cr.attractors
	}
	for _, a := range cr.attractors {
		if !a.Equals(attractor) {
			continue
		}
		attractors = append(attractors, a)
	}
	return attractors
}

// SetGateways sets the gateways and notifies the change via the channel
// with the resource type as Gateway
func (cr *ConfigurationRegistry) SetGateways(gateways []*nspAPI.Gateway) {
	cr.gateways = gateways
	cr.notify(&ConfigurationEvent{
		ResourceType: Gateway,
	})
}

// GetGateways returns gateways with the same name (ignored if empty) and
// the same parent as the one in parameter. If the gateway in parameter is
// nil, all gateways will be returned
func (cr *ConfigurationRegistry) GetGateways(gateway *nspAPI.Gateway) []*nspAPI.Gateway {
	gateways := []*nspAPI.Gateway{}
	if gateway == nil {
		return cr.gateways
	}
	for _, g := range cr.gateways {
		if !g.Equals(gateway) {
			continue
		}
		gateways = append(gateways, g)
	}
	return gateways
}

func (cr *ConfigurationRegistry) notify(configurationEvent *ConfigurationEvent) {
	if cr.Chan == nil {
		return
	}
	cr.Chan <- configurationEvent
}
