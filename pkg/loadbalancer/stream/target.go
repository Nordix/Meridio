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

type Target struct {
	fwMarks   []networking.FWMarkRoute
	nspTarget *nspAPI.Target
	netUtils  networking.Utils
}

func NewTarget(nspTarget *nspAPI.Target, netUtils networking.Utils) (types.Target, error) {
	target := &Target{
		fwMarks:   []networking.FWMarkRoute{},
		nspTarget: nspTarget,
		netUtils:  netUtils,
	}
	err := target.isValid()
	if err != nil {
		return nil, err
	}
	return target, nil
}

func (t *Target) GetIps() []string {
	return t.nspTarget.GetIps()
}

func (t *Target) GetIdentifier() int {
	return t.getIdentifier()
}

func (t *Target) Configure() error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
	}
	for _, ip := range t.GetIps() {
		var fwMark networking.FWMarkRoute
		fwMark, err := t.netUtils.NewFWMarkRoute(ip, t.GetIdentifier(), t.GetIdentifier())
		if err != nil {
			return err
		}
		t.fwMarks = append(t.fwMarks, fwMark)
	}
	return nil
}

func (t *Target) Delete() error {
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

func (t *Target) isValid() error {
	if t.nspTarget.GetStatus() != nspAPI.Target_ENABLED {
		return errors.New("the target is not enabled")
	}
	if t.getIdentifier() < 0 {
		return errors.New("identifier is not a number")
	}
	return nil
}

func (t *Target) getIdentifier() int {
	identifierStr, exists := t.nspTarget.GetContext()[types.IdentifierKey]
	if !exists {
		return -1
	}
	identifier, err := strconv.Atoi(identifierStr)
	if err != nil {
		return -1
	}
	return identifier
}
