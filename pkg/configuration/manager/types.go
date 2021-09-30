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

package manager

import nspAPI "github.com/nordix/meridio/api/nsp/v1"

type ConfigurationRegistry interface {
	GetTrench(*nspAPI.Trench) *nspAPI.Trench
	GetConduits(*nspAPI.Conduit) []*nspAPI.Conduit
	GetStreams(*nspAPI.Stream) []*nspAPI.Stream
	GetFlows(*nspAPI.Flow) []*nspAPI.Flow
	GetVips(*nspAPI.Vip) []*nspAPI.Vip
	GetAttractors(*nspAPI.Attractor) []*nspAPI.Attractor
	GetGateways(*nspAPI.Gateway) []*nspAPI.Gateway
}

type WatcherRegistry interface {
	RegisterWatcher(toWatch interface{}, ch interface{}) error
	UnregisterWatcher(ch interface{})
}
