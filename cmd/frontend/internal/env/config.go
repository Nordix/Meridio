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

package env

type Config struct {
	VRRPs             []string `default:"" desc:"VRRP IP addresses to be used as next-hops for static default routes" envconfig:"VRRPS"`
	ExternalInterface string   `default:"ext-vlan" desc:"External interface to start BIRD on" split_words:"true"`
	BirdConfigPath    string   `default:"/etc/bird" desc:"Path to place bird config files" split_words:"true"`
	LocalAS           string   `default:"8103" desc:"Local BGP AS number" envconfig:"LOCAL_AS"`
	RemoteAS          string   `default:"4248829953" desc:"Local BGP AS number" envconfig:"REMOTE_AS"`
	BGPLocalPort      string   `default:"10179" desc:"Local BGP server port" envconfig:"BGP_LOCAL_PORT"`
	BGPRemotePort     string   `default:"10179" desc:"Remote BGP server port" envconfig:"BGP_REMOTE_PORT"`
	BGPHoldTime       string   `default:"3" desc:"Seconds to wait for a Keepalive message from peer before considering the connection stale" envconfig:"BGP_HOLD_TIME"`
	TableID           int      `default:"4096" desc:"OS Kernel routing table ID BIRD syncs the routes with" envconfig:"TABLE_ID"`
	ECMP              bool     `default:"false" desc:"Enable ECMP towards next-hops of avaialble gateways" envconfig:"ECMP"`
	DropIfNoPeer      bool     `default:"false" desc:"Install default blackhole route with high metric into routing table TableID" split_words:"true"`
	LogBird           bool     `default:"false" desc:"Add important bird log snippets to our log" split_words:"true"`
	Namespace         string   `default:"default" desc:"Namespace the pod is running on" split_words:"true"`
	ConfigMapName     string   `default:"meridio-configuration" desc:"Name of the ConfigMap containing the configuration" split_words:"true"`
	NSPService        string   `default:"nsp-service-trench-a:7778" desc:"IP (or domain) and port of the NSP Service" split_words:"true"`
	TrenchName        string   `default:"default" desc:"Name of the Trench the frontend is associated with" split_words:"true"`
	AttractorName     string   `default:"default" desc:"Name of the Attractor the frontend is associated with" split_words:"true"`
}
