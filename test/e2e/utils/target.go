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

package utils

import (
	"strings"
)

type StreamStatus struct {
	Trench  string
	Conduit string
	Stream  string
	Status  string
}

func ParseTargetWatch(output string) []*StreamStatus {
	statusList := []*StreamStatus{}
	statusString := strings.Split(output, "New stream list:")
	if len(statusString) <= 1 {
		return statusList
	}
	status := strings.Split(statusString[1], "\n")
	for _, s := range status {
		if strings.Contains(s, "OPEN - ") || strings.Contains(s, "PENDING - ") || strings.Contains(s, "UNAVAILABLE - ") || strings.Contains(s, "UNDEFINED - ") {
			streamStatus := strings.Split(s, " - ")
			if len(streamStatus) <= 3 {
				continue
			}
			statusList = append(statusList, &StreamStatus{
				Trench:  streamStatus[3],
				Conduit: streamStatus[2],
				Stream:  streamStatus[1],
				Status:  streamStatus[0],
			})
		}
	}
	return statusList
}
