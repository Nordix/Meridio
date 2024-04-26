/*
Copyright (c) 2024 Nordix Foundation

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

import "sort"

func EqualStringList(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	tmpa := make([]string, len(a))
	tmpb := make([]string, len(b))
	copy(tmpa, tmpa)
	copy(tmpb, tmpb)
	copy(tmpa, a)
	copy(tmpb, b)
	sort.Strings(tmpa)
	sort.Strings(tmpb)

	for i := range tmpa {
		if tmpa[i] != tmpb[i] {
			return false
		}
	}
	return true
}
