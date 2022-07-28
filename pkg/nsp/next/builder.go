/*
Copyright (c) 2022 Nordix Foundation

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

package next

// BuildNextTargetRegistryChain chains the target registry servers together.
// Each NextTargetRegistryServer must have a non nil NextTargetRegistryServerImpl.
// If the list of nextTargetRegistryServers is nil or empty, a nil value will be returned.
func BuildNextTargetRegistryChain(nextTargetRegistryServers ...NextTargetRegistryServer) NextTargetRegistryServer {
	if len(nextTargetRegistryServers) <= 0 {
		return nil
	}
	for i, ntrs := range nextTargetRegistryServers {
		if i >= (len(nextTargetRegistryServers) - 1) {
			break
		}
		ntrs.setNext(nextTargetRegistryServers[i+1])
	}
	return nextTargetRegistryServers[0]
}
