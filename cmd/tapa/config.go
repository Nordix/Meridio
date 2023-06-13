/*
Copyright (c) 2021-2022 Nordix Foundation

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

import (
	"net/url"
	"time"
)

// Config for the TAPA
type Config struct {
	Name             string        `default:"nsc" desc:"Name of the target"`
	Node             string        `default:"" desc:"Node name the target is running on" split_words:"true"`
	Namespace        string        `default:"default" desc:"Namespace the trenches to connect to are running on" split_words:"true"`
	Socket           string        `default:"/ambassador.sock" desc:"Path of the socket file of the TAPA" split_words:"true"`
	NSMSocket        url.URL       `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"Path of the socket file of NSM" envconfig:"nsm_socket"`
	NSPServiceName   string        `default:"nsp-service" desc:"Domain name of the NSP Service" envconfig:"nsp_service_name"`
	NSPServicePort   int           `default:"7778" desc:"port of the NSP Service" envconfig:"nsp_service_port"`
	Timeout          time.Duration `default:"15s" desc:"timeout of NSM request/close, NSP register/unregister..." split_words:"true"`
	DialTimeout      time.Duration `default:"5s" desc:"timeout to dial NSMgr" split_words:"true"`
	MaxTokenLifetime time.Duration `default:"24h" desc:"maximum lifetime of tokens" split_words:"true"`
	LogLevel         string        `default:"DEBUG" desc:"Log level" split_words:"true"`
	NSPEntryTimeout  time.Duration `default:"30s" desc:"Timeout of the entries" envconfig:"nsp_entry_timeout"`
	GRPCMaxBackoff   time.Duration `default:"5s" desc:"Upper bound on gRPC connection backoff delay" envconfig:"grpc_max_backoff"`
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() error {
	return nil
}
