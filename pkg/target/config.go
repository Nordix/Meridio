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

package target

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
)

type Config struct {
	configMapName  string
	nspServiceName string
	nspServicePort int
	netUtils       networking.Utils
	nsmConfig      *nsm.Config
	apiClient      *nsm.APIClient
}

func NewConfig(configMapName string, nspServiceName string, nspServicePort int, netUtils networking.Utils, nsmConfig *nsm.Config) *Config {
	config := &Config{
		configMapName:  configMapName,
		nspServiceName: nspServiceName,
		nspServicePort: nspServicePort,
		netUtils:       netUtils,
		nsmConfig:      nsmConfig,
	}
	return config
}
