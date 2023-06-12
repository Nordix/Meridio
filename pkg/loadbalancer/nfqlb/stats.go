/*
Copyright (c) 2023 Nordix Foundation

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

package nfqlb

import (
	"encoding/json"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type FlowStats struct {
	Flow         *nspAPI.Flow
	MatchesCount int
}

func (fs *FlowStats) GetFlow() *nspAPI.Flow {
	return fs.Flow
}

func (fs *FlowStats) GetMatchesCount() int {
	return fs.MatchesCount
}

// nfqlbFlowStats represents the nfqlb format returned with
// nfqlb flow-list
type nfqlbFlowStats struct {
	Name         string `json:"Name"`
	UserRef      string `json:"user_ref"`
	MatchesCount int    `json:"matches_count"`
}

func (nfqlbFw *nfqlbFlowStats) getFlowName() string {
	return nfqlbFw.Name[len(nfqlbFw.UserRef)+1:] // e.g. "tshm-stream-a-i" + "-"
}

func (nfqlbFw *nfqlbFlowStats) getStreamName() string {
	return nfqlbFw.UserRef[len("tshm-"):]
}

// GetFlowStats returns the list of currently configured flows in
// nfqlb together with their match count metric.
// The flow will contain only the flow name and stream name since
// nfqlb is not aware about parent names (conduit/trench).
func GetFlowStats() ([]*FlowStats, error) {
	fs := []*FlowStats{}
	jsonStatsStr, err := FlowList()
	if err != nil {
		return nil, err
	}
	nfqlbFss := []*nfqlbFlowStats{}
	err = json.Unmarshal([]byte(jsonStatsStr), &nfqlbFss)
	if err != nil {
		return nil, err
	}
	for _, nfqlbFs := range nfqlbFss {
		fs = append(fs, &FlowStats{
			Flow: &nspAPI.Flow{
				Name: nfqlbFs.getFlowName(),
				Stream: &nspAPI.Stream{
					Name: nfqlbFs.getStreamName(),
				},
			},
			MatchesCount: nfqlbFs.MatchesCount,
		})
	}
	return fs, nil
}
