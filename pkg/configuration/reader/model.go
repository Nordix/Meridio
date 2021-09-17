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

type Trench struct {
	Name string `yaml:"name"`
}

type ConduitList struct {
	Conduits []*Conduit `yaml:"items"`
}

type Conduit struct {
	Name   string `yaml:"name"`
	Trench string `yaml:"trench"`
}

type StreamList struct {
	Streams []*Stream `yaml:"items"`
}

type Stream struct {
	Name    string `yaml:"name"`
	Conduit string `yaml:"conduit"`
}

type FlowList struct {
	Flows []*Flow `yaml:"items"`
}

type Flow struct {
	Name                  string   `yaml:"name"`
	SourceSubnets         []string `yaml:"source-subnets"`
	DestinationPortRanges []string `yaml:"destination-port-ranges"`
	SourcePortRanges      []string `yaml:"source-port-ranges"`
	Protocols             []string `yaml:"protocols"`
	Vips                  []string `yaml:"vips"`
	Stream                string   `yaml:"stream"`
}

type VipList struct {
	Vips []*Vip `yaml:"items"`
}

type Vip struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	Trench  string `yaml:"trench"`
}

type AttractorList struct {
	Attractors []*Attractor `yaml:"items"`
}

type Attractor struct {
	Name     string   `yaml:"name"`
	Vips     []string `yaml:"vips"`
	Gateways []string `yaml:"gateways"`
	Trench   string   `yaml:"trench"`
}

type GatewayList struct {
	Gateways []*Gateway `yaml:"items"`
}

type Gateway struct {
	Name       string `yaml:"name"`
	Address    string `yaml:"address"`
	RemoteASN  uint32 `yaml:"remote-asn"`
	LocalASN   uint32 `yaml:"local-asn"`
	RemotePort uint16 `yaml:"remote-port"`
	LocalPort  uint16 `yaml:"local-port"`
	IPFamily   string `yaml:"ip-family"`
	BFD        bool   `yaml:"bfd"`
	Protocol   string `yaml:"protocol"`
	HoldTime   uint   `yaml:"hold-time"`
	Trench     string `yaml:"trench"`
}
