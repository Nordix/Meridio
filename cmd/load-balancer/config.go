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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config for the proxy
type Config struct {
	Name             string        `default:"load-balancer" desc:"Name of the pod"`
	ServiceName      string        `default:"load-balancer" desc:"Name of providing service" split_words:"true"`
	ConnectTo        url.URL       `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to NSM" split_words:"true"`
	DialTimeout      time.Duration `default:"5s" desc:"timeout to dial NSMgr" split_words:"true"`
	RequestTimeout   time.Duration `default:"15s" desc:"timeout to request NSE" split_words:"true"`
	MaxTokenLifetime time.Duration `default:"24h" desc:"maximum lifetime of tokens" split_words:"true"`
	NSPService       string        `default:"nsp-service:7778" desc:"IP (or domain) and port of the NSP Service" split_words:"true"`
	ConduitName      string        `default:"load-balancer" desc:"Name of the conduit" split_words:"true"`
	TrenchName       string        `default:"default" desc:"Trench the pod is running on" split_words:"true"`
	LogLevel         string        `default:"DEBUG" desc:"Log level" split_words:"true"`
	Nfqueue          string        `default:"0:3" desc:"netfilter queue(s) to be used by nfqlb" split_words:"true"`
	NfqueueFanout    bool          `default:"false" desc:"enable fanout nfqueue option" split_words:"true"`

	ProfilingEnabled bool `default:"false" desc:"enable profiling" split_words:"true"`
	ProfilingPort    int  `default:"9995" desc:"port of the profiling http server" split_words:"true"`
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() error {
	if c.ConnectTo.String() == "" {
		return errors.New("no NSMGr ConnectTO URL are specified")
	}

	nfq := strings.Split(c.Nfqueue, ":")
	if _, err := strconv.ParseUint(nfq[0], 10, 16); err != nil {
		return fmt.Errorf("wrong Nfqueue format; %v (%v)", nfq[0], err)
	}
	if len(nfq) >= 2 {
		if _, err := strconv.ParseUint(nfq[1], 10, 16); err != nil {
			return fmt.Errorf("wrong Nfqueue format; %v (%v)", nfq[1], err)
		}
	}

	return nil
}
