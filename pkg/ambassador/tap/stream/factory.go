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

package stream

import (
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
)

type streamFactory struct {
	TargetRegistryClient nspAPI.TargetRegistryClient
	Conduit              Conduit
}

func NewFactory(targetRegistryClient nspAPI.TargetRegistryClient,
	conduit Conduit) *streamFactory {
	sfi := &streamFactory{
		TargetRegistryClient: targetRegistryClient,
		Conduit:              conduit,
	}
	return sfi
}

func (sf *streamFactory) New(strm *ambassadorAPI.Stream) (types.Stream, error) {
	return New(strm, sf.TargetRegistryClient, sf.Conduit)
}
