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

package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	targetMetrics "github.com/nordix/meridio/pkg/loadbalancer/target"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/utils"
)

type target struct {
	fwMarks           []networking.FWMarkRoute
	nspTarget         *nspAPI.Target
	netUtils          networking.Utils
	identifier        int
	identifierOffset  int
	targetHitsMetrics *targetMetrics.HitsMetrics
}

func NewTarget(nspTarget *nspAPI.Target, netUtils networking.Utils, targetHitsMetrics *targetMetrics.HitsMetrics, identifierOffset int) (types.Target, error) {
	target := &target{
		fwMarks:           []networking.FWMarkRoute{},
		nspTarget:         nspTarget,
		netUtils:          netUtils,
		targetHitsMetrics: targetHitsMetrics,
		identifierOffset:  identifierOffset,
	}
	if nspTarget.GetStatus() != nspAPI.Target_ENABLED {
		return nil, errors.New("the target is not enabled")
	}
	idStr, exists := nspTarget.GetContext()[types.IdentifierKey]
	if !exists {
		return nil, fmt.Errorf("no identifier")
	}
	var err error
	target.identifier, err = strconv.Atoi(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid identifier: %w", err)
	}
	return target, nil
}

func (t *target) GetIps() []string {
	return t.nspTarget.GetIps()
}

func (t *target) GetIdentifier() int {
	return t.identifier
}

func (t *target) Verify() bool {
	for _, fwMark := range t.fwMarks {
		if !fwMark.Verify() {
			return false
		}
	}
	return true
}

func (t *target) Configure() error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
	}
	offsetId := t.identifier + t.identifierOffset
	for _, ip := range t.GetIps() {
		var fwMark networking.FWMarkRoute
		fwMark, err := t.netUtils.NewFWMarkRoute(ip, offsetId, offsetId)
		if err != nil {
			return fmt.Errorf("failed to configure fwmark route for ip (%s): %w", ip, err)
		}
		t.fwMarks = append(t.fwMarks, fwMark)
	}
	_ = t.targetHitsMetrics.Register(offsetId, t.nspTarget)
	return nil
}

func (t *target) Delete() error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
		return nil
	}
	var errFinal error
	for _, fwMark := range t.fwMarks {
		err := fwMark.Delete()
		if err != nil {
			errFinal = utils.AppendErr(errFinal, fmt.Errorf("fwMark delete: %w", err)) // todo
		}
	}
	t.fwMarks = []networking.FWMarkRoute{}
	offsetId := t.identifier + t.identifierOffset
	_ = t.targetHitsMetrics.Unregister(offsetId)
	return errFinal
}

func (t *target) MarshalJSON() ([]byte, error) {
	ts := struct {
		Identifier int      `json:"identifier"`
		IPs        []string `json:"ips"`
	}{
		t.identifier,
		t.nspTarget.GetIps(),
	}
	enc, err := json.Marshal(&ts)
	if err != nil {
		return enc, fmt.Errorf("failed to marshal target (%v): %w", ts, err)
	}
	return enc, nil
}
