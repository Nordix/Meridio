/*
Copyright (c) 2025 OpenInfra Foundation Europe

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

// MergeMapsInPlace returns a merged map based on the existing map extended or
// overwritten based on the desired map.
// This function can be used for both labels and annotations.
func MergeMapsInPlace(existing map[string]string, desired map[string]string) map[string]string {
	if len(desired) == 0 {
		return existing // empty desired map, return existing (could be nil or empty)
	}

	if len(existing) == 0 {
		return desired // empty existing map, return desired (could NOT be nil or empty, otherwise the condition above would have been met)
	}

	// add missing and overwrite existing entries based on the desired values
	for key, value := range desired {
		existing[key] = value
	}
	return existing
}
