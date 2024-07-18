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

package client

import (
	"errors"
	"net/url"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/nsm"
)

// Config - configuration for network service client
type Config struct {
	Name                    string
	RequestTimeout          time.Duration
	ConnectTo               url.URL
	MaxTokenLifetime        time.Duration
	APIClient               *nsm.APIClient
	MonitorConnectionClient networkservice.MonitorConnectionClient
}

// IsValid - check if configuration is valid
func (c *Config) IsValid() error {
	if c.Name == "" {
		return errors.New("no client name specified")
	}
	return nil
}
