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

package registry

import (
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
)

func Filter(stream *ambassadorAPI.Stream, streams []*ambassadorAPI.StreamStatus) []*ambassadorAPI.StreamStatus {
	if stream == nil {
		return streams
	}
	result := []*ambassadorAPI.StreamStatus{}
	for _, s := range streams {
		if ambassadorAPI.StreamFilter(stream, s.Stream) {
			result = append(result, s)
		}
	}
	return result
}
