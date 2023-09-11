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

import "time"

type Config struct {
	VRRPs                 []string      `default:"" desc:"VRRP IP addresses to be used as next-hops for static default routes" envconfig:"VRRPS"`
	ExternalInterface     string        `default:"ext-vlan" desc:"External interface to start BIRD on" split_words:"true"`
	BirdConfigPath        string        `default:"/etc/bird" desc:"Path to place bird config files" split_words:"true"`
	BirdCommunicationSock string        `default:"/var/run/bird/bird.ctl" desc:"Use given filename for a socket to communicate with birdc" split_words:"true"`
	BirdLogFileSize       int           `default:"20000" desc:"File size in bytes of the local BIRD log file (and log backup file)" split_words:"true"`
	LocalAS               string        `default:"8103" desc:"Local BGP AS number" envconfig:"LOCAL_AS"`
	RemoteAS              string        `default:"4248829953" desc:"Local BGP AS number" envconfig:"REMOTE_AS"`
	BGPLocalPort          string        `default:"10179" desc:"Local BGP server port" envconfig:"BGP_LOCAL_PORT"`
	BGPRemotePort         string        `default:"10179" desc:"Remote BGP server port" envconfig:"BGP_REMOTE_PORT"`
	BGPHoldTime           string        `default:"3" desc:"Seconds to wait for a Keepalive message from peer before considering the connection stale" envconfig:"BGP_HOLD_TIME"`
	TableID               int           `default:"4096" desc:"Start ID of the two consecutive OS Kernel routing tables BIRD syncs the routes with" envconfig:"TABLE_ID"`
	ECMP                  bool          `default:"false" desc:"Enable ECMP towards next-hops of avaialble gateways" envconfig:"ECMP"`
	DropIfNoPeer          bool          `default:"true" desc:"Install default blackhole route with high metric into routing table TableID" split_words:"true"`
	LogBird               bool          `default:"false" desc:"Add important bird log snippets to our log" split_words:"true"`
	Namespace             string        `default:"default" desc:"Namespace the pod is running on" split_words:"true"`
	NSPService            string        `default:"nsp-service-trench-a:7778" desc:"IP (or domain) and port of the NSP Service" split_words:"true"`
	TrenchName            string        `default:"default" desc:"Name of the Trench the frontend is associated with" split_words:"true"`
	AttractorName         string        `default:"default" desc:"Name of the Attractor the frontend is associated with" split_words:"true"`
	LogLevel              string        `default:"DEBUG" desc:"Log level" split_words:"true"`
	NSPEntryTimeout       time.Duration `default:"30s" desc:"Timeout of the entries" envconfig:"nsp_entry_timeout"`
	GRPCKeepaliveTime     time.Duration `default:"30s" desc:"gRPC keepalive timeout"`
	GRPCMaxBackoff        time.Duration `default:"5s" desc:"Upper bound on gRPC connection backoff delay" envconfig:"grpc_max_backoff"`
	DelayConnectivity     time.Duration `default:"1s" desc:"Delay between checks with connectivity"`
	DelayNoConnectivity   time.Duration `default:"3s" desc:"Delay between checks without connectivity"`
	MaxSessionErrors      int           `default:"5" desc:"Max session errors when checking Bird until denounce"`
}
