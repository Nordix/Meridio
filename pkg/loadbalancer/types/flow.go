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

package types

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type Flow interface {
	Update(*nspAPI.Flow) error
	Delete() error
}

type NftHandler interface {
	PortNATSet(flowName string, protocols []string, dport, localPort uint) error
	PortNATDelete(flowName string)
	PortNATCreateSets(flow *nspAPI.Flow) error
	PortNATDeleteSets(flow *nspAPI.Flow)
	PortNATSetAddresses(flow *nspAPI.Flow) error
}
