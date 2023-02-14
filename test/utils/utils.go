/*
Copyright (c) 2021-2023 Nordix Foundation

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

package utils

import (
	"github.com/nordix/meridio/pkg/configuration/reader"
)

// Input is a slice of Vips.
// Return a map with key as vip names.
func MakeMapFromVipList(lst []*reader.Vip) map[string]reader.Vip {
	ret := make(map[string]reader.Vip)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}

// Input is a slice of Gateways.
// Return a map with key as gateway names.
func MakeMapFromGWList(lst []*reader.Gateway) map[string]reader.Gateway {
	ret := make(map[string]reader.Gateway)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}

// Input is a slice of Attractors.
// Return a map with key as attractor names.
func MakeMapFromAttractorList(lst []*reader.Attractor) map[string]reader.Attractor {
	ret := make(map[string]reader.Attractor)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}

// Input is a slice of Conduits.
// Return a map with key as conduit names.
func MakeMapFromConduitList(lst []*reader.Conduit) map[string]reader.Conduit {
	ret := make(map[string]reader.Conduit)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}

// Input is a slice of Streams.
// Return a map with key as stream names.
func MakeMapFromStreamList(lst []*reader.Stream) map[string]reader.Stream {
	ret := make(map[string]reader.Stream)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}

// Input is a slice of Flows.
// Return a map with key as flow names.
func MakeMapFromFlowList(lst []*reader.Flow) map[string]reader.Flow {
	ret := make(map[string]reader.Flow)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}
