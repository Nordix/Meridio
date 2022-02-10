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

import (
	"errors"
	"net/url"
	"time"
)

// Config for the proxy
type Config struct {
	Name               string        `default:"proxy" desc:"Pod Name"`
	ServiceName        string        `default:"proxy" desc:"Name of the Network Service" split_words:"true"`
	ConnectTo          url.URL       `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to NSM" split_words:"true"`
	DialTimeout        time.Duration `default:"5s" desc:"timeout to dial NSMgr" split_words:"true"`
	RequestTimeout     time.Duration `default:"15s" desc:"timeout to request NSE" split_words:"true"`
	MaxTokenLifetime   time.Duration `default:"24h" desc:"maximum lifetime of tokens" split_words:"true"`
	IPAMService        string        `default:"ipam-service:7777" desc:"IP (or domain) and port of the IPAM Service" split_words:"true"`
	Host               string        `default:"" desc:"Host name the proxy is running on" split_words:"true"`
	NetworkServiceName string        `default:"load-balancer" desc:"Name of the network service the proxy request the connection" split_words:"true"`
	Namespace          string        `default:"default" desc:"Namespace the pod is running on" split_words:"true"`
	Trench             string        `default:"default" desc:"Trench the pod is running on" split_words:"true"`
	Conduit            string        `default:"load-balancer" desc:"Name of the conduit" split_words:"true"`
	NSPServiceName     string        `default:"nsp-service" desc:"IP (or domain) of the NSP Service" split_words:"true"`
	NSPServicePort     int           `default:"7778" desc:"port of the NSP Service" split_words:"true"`
	IPFamily           string        `default:"dualstack" desc:"ip family" envconfig:"ip_family"`
	LogLevel           string        `default:"DEBUG" desc:"Log level" split_words:"true"`
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() error {
	if c.ConnectTo.String() == "" {
		return errors.New("no NSMGr ConnectTO URL are specified")
	}
	return nil
}
