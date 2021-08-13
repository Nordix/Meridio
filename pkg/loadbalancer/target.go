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

package loadbalancer

import "github.com/nordix/meridio/pkg/networking"

type Target struct {
	identifier int
	ips        []string
	fwMarks    []networking.FWMarkRoute
}

func (t *Target) GetIdentifier() int {
	return t.identifier
}

func (t *Target) GetIPs() []string {
	return t.ips
}

func (t *Target) Configure(netUtils networking.Utils) error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
	}
	var err error
	for _, ip := range t.ips {
		var fwMark networking.FWMarkRoute
		fwMark, err = netUtils.NewFWMarkRoute(ip, t.identifier, t.identifier)
		t.fwMarks = append(t.fwMarks, fwMark)
	}
	return err
}

func (t *Target) Delete() error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
		return nil
	}
	var err error
	for _, fwMark := range t.fwMarks {
		err = fwMark.Delete()
	}
	t.fwMarks = []networking.FWMarkRoute{}
	return err
}

func NewTarget(identifier int, ips []string) *Target {
	target := &Target{
		identifier: identifier,
		ips:        ips,
		fwMarks:    []networking.FWMarkRoute{},
	}
	return target
}
