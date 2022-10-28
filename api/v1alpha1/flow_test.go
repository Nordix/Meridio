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

package v1alpha1

import (
	"testing"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		desc  string
		ports []string
		valid bool
	}{
		{
			desc: "overlapping ports 1",
			ports: []string{
				"2000-3000",
				"2010-3000",
			},
			valid: false,
		},
		{
			desc: "overlapping ports 2",
			ports: []string{
				"2000-3000",
				"2010",
			},
			valid: false,
		},
		{
			desc: "overlapping ports 3",
			ports: []string{
				"2000-3000",
				"1000-2000",
				"1500-2500",
			},
			valid: false,
		},
		{
			desc: "overlapping ports 4",
			ports: []string{
				"any",
				"1000-2000",
			},
			valid: false,
		},
		{
			desc: "wrong port format 1",
			ports: []string{
				"2000-3000",
				"2010-2",
			},
			valid: false,
		},
		{
			desc: "wrong port format 2",
			ports: []string{
				"2000-3000",
				"2010-a",
			},
			valid: false,
		},
		{
			desc: "valid format 1",
			ports: []string{
				"2000-3000",
				"1000-1500",
				"4500-5000",
			},
			valid: true,
		},
		{
			desc: "valid format 2",
			ports: []string{
				"2000",
				"1000",
				"4500-5000",
			},
			valid: true,
		},
		{
			desc: "valid format 3",
			ports: []string{
				"2000",
				"1000",
			},
			valid: true,
		},
	}
	for _, test := range tests {
		_, err := validatePorts(test.ports)
		if (err == nil) != test.valid {
			t.Errorf("case: %s validation failed", test.desc)
		}
	}
}

func TestValidateSubnet(t *testing.T) {
	tests := []struct {
		desc    string
		subnets []string
		valid   bool
	}{
		{
			desc: "subnet wrong format",
			subnets: []string{
				"10.0.0.1/28",
			},
			valid: false,
		},
		{
			desc: "overlapping subnets 1",
			subnets: []string{
				"10.0.0.0/28",
				"10.0.0.2/31",
			},
			valid: false,
		},
		{
			desc: "overlapping subnets 2",
			subnets: []string{
				"10.0.0.0/28",
				"20.0.0.2/31",
				"10.0.0.0/20",
			},
			valid: false,
		},
		{
			desc: "valid subnets",
			subnets: []string{
				"10.0.0.0/28",
				"1000::/126",
				"20.0.0.2/31",
			},
			valid: true,
		},
	}
	for _, test := range tests {
		_, err := validateSubnets(test.subnets)
		if (err == nil) != test.valid {
			t.Errorf("case: %s validation failed", test.desc)
		}
	}
}
