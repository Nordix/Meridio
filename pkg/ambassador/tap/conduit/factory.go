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

package conduit

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/stream"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
)

type PendingChanFunc func() <-chan interface{}

type StreamFactory interface {
	New(*nspAPI.Stream, stream.Conduit) (types.Stream, error)
}

type streamFactoryImpl struct {
	TargetRegistryClient       nspAPI.TargetRegistryClient
	MaxNumberOfTargets         int
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	StreamRegistry             types.Registry
	PendingChanFunc            PendingChanFunc
}

func newStreamFactoryImpl(targetRegistryClient nspAPI.TargetRegistryClient,
	configurationManagerClient nspAPI.ConfigurationManagerClient,
	streamRegistry types.Registry,
	maxNumberOfTargets int,
	pendingChanFunc PendingChanFunc) *streamFactoryImpl {
	sfi := &streamFactoryImpl{
		TargetRegistryClient:       targetRegistryClient,
		MaxNumberOfTargets:         maxNumberOfTargets,
		ConfigurationManagerClient: configurationManagerClient,
		StreamRegistry:             streamRegistry,
		PendingChanFunc:            pendingChanFunc,
	}
	return sfi
}

func (sfi *streamFactoryImpl) New(strm *nspAPI.Stream, cndt stream.Conduit) (types.Stream, error) {
	return stream.New(strm, sfi.TargetRegistryClient, sfi.ConfigurationManagerClient, sfi.StreamRegistry, sfi.MaxNumberOfTargets, sfi.PendingChanFunc(), cndt)
}