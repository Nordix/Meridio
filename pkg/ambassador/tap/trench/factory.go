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

package trench

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/conduit"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/networking"
)

type ConduitFactory interface {
	New(*nspAPI.Conduit) (types.Conduit, error)
}

type conduitFactoryImpl struct {
	TargetName                 string
	Namespace                  string
	NodeName                   string
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	TargetRegistryClient       nspAPI.TargetRegistryClient
	NetworkServiceClient       networkservice.NetworkServiceClient
	StreamRegistry             types.Registry
	NetUtils                   networking.Utils
}

func newConduitFactoryImpl(
	targetName string,
	namespace string,
	nodeName string,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	targetRegistryClient nspAPI.TargetRegistryClient,
	networkServiceClient networkservice.NetworkServiceClient,
	streamRegistry types.Registry,
	netUtils networking.Utils) *conduitFactoryImpl {
	cfi := &conduitFactoryImpl{
		TargetName:                 targetName,
		Namespace:                  namespace,
		NodeName:                   nodeName,
		ConfigurationManagerClient: configurationManagerClient,
		TargetRegistryClient:       targetRegistryClient,
		NetworkServiceClient:       networkServiceClient,
		StreamRegistry:             streamRegistry,
		NetUtils:                   netUtils,
	}
	return cfi
}

func (cfi *conduitFactoryImpl) New(cndt *nspAPI.Conduit) (types.Conduit, error) {
	return conduit.New(cndt,
		cfi.TargetName,
		cfi.Namespace,
		cfi.NodeName,
		cfi.ConfigurationManagerClient,
		cfi.TargetRegistryClient,
		cfi.NetworkServiceClient,
		cfi.StreamRegistry,
		cfi.NetUtils)
}
