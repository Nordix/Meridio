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

package main

// Config for the ipam
type Config struct {
	Port                    int    `default:"7777" desc:"Trench the pod is running on" split_words:"true"`
	Datasource              string `default:"/run/ipam/data/registry.db" desc:"Path and file name of the sqlite database" split_words:"true"`
	TrenchName              string `default:"default" desc:"Trench the pod is running on" split_words:"true"`
	NSPService              string `default:"nsp-service:7778" desc:"IP (or domain) and port of the NSP Service" split_words:"true"`
	PrefixIPv4              string `default:"169.255.0.0/16" desc:"ipv4 prefix from which the proxy prefixes will be allocated" envconfig:"prefix_ipv4"`
	ConduitPrefixLengthIPv4 int    `default:"20" desc:"conduit prefix length which will be allocated" envconfig:"conduit_prefix_length_ipv4"`
	NodePrefixLengthIPv4    int    `default:"24" desc:"node prefix length which will be allocated" envconfig:"node_prefix_length_ipv4"`
	PrefixIPv6              string `default:"fd00::/48" desc:"ipv4 prefix from which the proxy prefixes will be allocated" envconfig:"prefix_ipv6"`
	ConduitPrefixLengthIPv6 int    `default:"56" desc:"conduit prefix length which will be allocated" envconfig:"conduit_prefix_length_ipv6"`
	NodePrefixLengthIPv6    int    `default:"64" desc:"node prefix length which will be allocated" envconfig:"node_prefix_length_ipv6"`
	IPFamily                string `default:"dualstack" desc:"ip family" envconfig:"ip_family"`
	LogLevel                string `default:"DEBUG" desc:"Log level" split_words:"true"`

	ProfilingEnabled bool `default:"false" desc:"enable profiling" split_words:"true"`
	ProfilingPort    int  `default:"9995" desc:"port of the profiling http server" split_words:"true"`
}
