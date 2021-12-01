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

package common

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

func Filter(target *nspAPI.Target, targets []*nspAPI.Target) []*nspAPI.Target {
	if target == nil {
		return targets
	}
	result := []*nspAPI.Target{}
	for _, t := range targets {
		if nspAPI.TargetFilter(target, t) {
			result = append(result, t)
		}
	}
	return result
}
