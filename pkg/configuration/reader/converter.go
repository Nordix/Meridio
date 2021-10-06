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
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

func ConvertAll(
	trench *Trench,
	conduits []*Conduit,
	streams []*Stream,
	flows []*Flow,
	vips []*Vip,
	attractors []*Attractor,
	gateways []*Gateway,
) (
	*nspAPI.Trench,
	[]*nspAPI.Conduit,
	[]*nspAPI.Stream,
	[]*nspAPI.Flow,
	[]*nspAPI.Vip,
	[]*nspAPI.Attractor,
	[]*nspAPI.Gateway,
) {
	trenchConverted := ConvertTrench(trench)
	conduitsConverted := ConvertConduits(conduits, trenchConverted)
	streamsConverted := ConvertStreams(streams, conduitsConverted)
	vipsConverted := ConvertVips(vips, trenchConverted)
	flowsConverted := ConvertFlows(flows, streamsConverted, vipsConverted)
	gatewaysConverted := ConvertGateways(gateways, trenchConverted)
	attractorsConverted := ConvertAttractors(attractors, trenchConverted, vipsConverted, gatewaysConverted)
	return trenchConverted, conduitsConverted, streamsConverted, flowsConverted, vipsConverted, attractorsConverted, gatewaysConverted
}

func ConvertTrench(trench *Trench) *nspAPI.Trench {
	return &nspAPI.Trench{
		Name: trench.Name,
	}
}

func ConvertConduits(conduits []*Conduit, trench *nspAPI.Trench) []*nspAPI.Conduit {
	resConduits := []*nspAPI.Conduit{}
	if trench == nil {
		return resConduits
	}
	for _, conduit := range conduits {
		if conduit.Trench != trench.Name {
			continue
		}
		resConduits = append(resConduits, &nspAPI.Conduit{
			Name:   conduit.Name,
			Trench: trench,
		})
	}
	return resConduits
}

func ConvertStreams(streams []*Stream, conduits []*nspAPI.Conduit) []*nspAPI.Stream {
	conduitsMap := make(map[string]*nspAPI.Conduit)
	for _, conduit := range conduits {
		conduitsMap[conduit.Name] = conduit
	}
	resStreams := []*nspAPI.Stream{}
	for _, stream := range streams {
		c, conduitExists := conduitsMap[stream.Conduit]
		if !conduitExists {
			continue
		}
		resStreams = append(resStreams, &nspAPI.Stream{
			Name:    stream.Name,
			Conduit: c,
		})
	}
	return resStreams
}

func ConvertFlows(flows []*Flow, streams []*nspAPI.Stream, vips []*nspAPI.Vip) []*nspAPI.Flow {
	streamsMap := make(map[string]*nspAPI.Stream)
	for _, stream := range streams {
		streamsMap[stream.Name] = stream
	}
	resFlows := []*nspAPI.Flow{}
	for _, flow := range flows {
		s, streamExists := streamsMap[flow.Stream]
		if !streamExists {
			continue
		}
		resFlows = append(resFlows, &nspAPI.Flow{
			Name:                  flow.Name,
			SourceSubnets:         flow.SourceSubnets,
			DestinationPortRanges: flow.DestinationPortRanges,
			SourcePortRanges:      flow.SourcePortRanges,
			Protocols:             flow.Protocols,
			Stream:                s,
			Vips:                  getVips(flow.Vips, vips),
		})
	}
	return resFlows
}

func ConvertVips(vips []*Vip, trench *nspAPI.Trench) []*nspAPI.Vip {
	resVips := []*nspAPI.Vip{}
	if trench == nil {
		return resVips
	}
	for _, vip := range vips {
		if vip.Trench != trench.Name {
			continue
		}
		resVips = append(resVips, &nspAPI.Vip{
			Name:    vip.Name,
			Address: vip.Address,
			Trench:  trench,
		})
	}
	return resVips
}

func ConvertAttractors(attractors []*Attractor, trench *nspAPI.Trench, vips []*nspAPI.Vip, gateways []*nspAPI.Gateway) []*nspAPI.Attractor {
	resAttractors := []*nspAPI.Attractor{}
	if trench == nil {
		return resAttractors
	}
	for _, attractor := range attractors {
		if attractor.Trench != trench.Name {
			continue
		}
		resAttractors = append(resAttractors, &nspAPI.Attractor{
			Name:     attractor.Name,
			Trench:   trench,
			Vips:     getVips(attractor.Vips, vips),
			Gateways: getGateways(attractor.Gateways, gateways),
		})
	}
	return resAttractors
}

func ConvertGateways(gateways []*Gateway, trench *nspAPI.Trench) []*nspAPI.Gateway {
	resGateways := []*nspAPI.Gateway{}
	if trench == nil {
		return resGateways
	}
	for _, gateway := range gateways {
		if gateway.Trench != trench.Name {
			continue
		}
		resGateways = append(resGateways, &nspAPI.Gateway{
			Name:       gateway.Name,
			Address:    gateway.Address,
			RemoteASN:  gateway.RemoteASN,
			LocalASN:   gateway.LocalASN,
			RemotePort: uint32(gateway.RemotePort),
			LocalPort:  uint32(gateway.LocalPort),
			IpFamily:   gateway.IPFamily,
			Bfd:        gateway.BFD,
			Protocol:   gateway.Protocol,
			HoldTime:   uint32(gateway.HoldTime),
			Trench:     trench,
		})
	}
	return resGateways
}

func getVips(vips []string, vipsAPI []*nspAPI.Vip) []*nspAPI.Vip {
	vipsAPIMap := make(map[string]*nspAPI.Vip)
	for _, vip := range vipsAPI {
		vipsAPIMap[vip.Name] = vip
	}
	resVips := []*nspAPI.Vip{}
	for _, v := range vips {
		vip, vipExists := vipsAPIMap[v]
		if !vipExists {
			continue
		}
		resVips = append(resVips, vip)
	}
	return resVips
}

func getGateways(gateways []string, gatewaysAPI []*nspAPI.Gateway) []*nspAPI.Gateway {
	gatewaysAPIMap := make(map[string]*nspAPI.Gateway)
	for _, gateway := range gatewaysAPI {
		gatewaysAPIMap[gateway.Name] = gateway
	}
	resGateways := []*nspAPI.Gateway{}
	for _, g := range gateways {
		gateway, gatewayExists := gatewaysAPIMap[g]
		if !gatewayExists {
			continue
		}
		resGateways = append(resGateways, gateway)
	}
	return resGateways
}
