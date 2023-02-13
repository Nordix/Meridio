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
	"errors"
	"fmt"
	"strconv"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
)

type target struct {
	fwMarks    []networking.FWMarkRoute
	nspTarget  *nspAPI.Target
	netUtils   networking.Utils
	identifier int
}

func NewTarget(nspTarget *nspAPI.Target, netUtils networking.Utils) (types.Target, error) {
	target := &target{
		fwMarks:   []networking.FWMarkRoute{},
		nspTarget: nspTarget,
		netUtils:  netUtils,
	}
	if nspTarget.GetStatus() != nspAPI.Target_ENABLED {
		return nil, errors.New("the target is not enabled")
	}
	idStr, exists := nspTarget.GetContext()[types.IdentifierKey]
	if !exists {
		return nil, fmt.Errorf("No identifier")
	}
	var err error
	target.identifier, err = strconv.Atoi(idStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid identifier: %w", err)
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

func (t *target) Configure(identifierOffset int) error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
	}
	for _, ip := range t.GetIps() {
		var fwMark networking.FWMarkRoute
		offsetId := t.identifier + identifierOffset
		fwMark, err := t.netUtils.NewFWMarkRoute(ip, offsetId, offsetId)
		if err != nil {
			return err
		}
		t.fwMarks = append(t.fwMarks, fwMark)
	}
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
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}
	}
	t.fwMarks = []networking.FWMarkRoute{}
	return errFinal
}
