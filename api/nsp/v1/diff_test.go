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

package v1

import (
	reflect "reflect"
	"testing"
)

func TestPortNatDiff(t *testing.T) {
	type args struct {
		old []*Conduit_PortNat
		new []*Conduit_PortNat
	}
	tests := []struct {
		name  string
		args  args
		want  []*Conduit_PortNat
		want1 []*Conduit_PortNat
		want2 []*Conduit_PortNat
	}{
		{
			name: "empty",
			args: args{
				old: []*Conduit_PortNat{},
				new: []*Conduit_PortNat{},
			},
			want:  []*Conduit_PortNat{},
			want1: []*Conduit_PortNat{},
			want2: []*Conduit_PortNat{},
		},
		{
			name: "1 added",
			args: args{
				old: []*Conduit_PortNat{},
				new: []*Conduit_PortNat{
					{
						Port:       10,
						TargetPort: 10,
						Protocol:   "TCP",
						Vips: []*Vip{
							{
								Name:    "vip-a",
								Address: "20.0.0.1/32",
							},
						},
					},
				},
			},
			want: []*Conduit_PortNat{{
				Port:       10,
				TargetPort: 10,
				Protocol:   "TCP",
				Vips: []*Vip{
					{
						Name:    "vip-a",
						Address: "20.0.0.1/32",
					},
				},
			}},
			want1: []*Conduit_PortNat{},
			want2: []*Conduit_PortNat{},
		},
		{
			name: "1 removed",
			args: args{
				old: []*Conduit_PortNat{
					{
						Port:       10,
						TargetPort: 10,
						Protocol:   "TCP",
						Vips: []*Vip{
							{
								Name:    "vip-a",
								Address: "20.0.0.1/32",
							},
						},
					},
				},
				new: []*Conduit_PortNat{},
			},
			want:  []*Conduit_PortNat{},
			want1: []*Conduit_PortNat{},
			want2: []*Conduit_PortNat{
				{
					Port:       10,
					TargetPort: 10,
					Protocol:   "TCP",
					Vips: []*Vip{
						{
							Name:    "vip-a",
							Address: "20.0.0.1/32",
						},
					},
				},
			},
		},
		{
			name: "1 common with different vips",
			args: args{
				old: []*Conduit_PortNat{
					{
						Port:       10,
						TargetPort: 10,
						Protocol:   "TCP",
						Vips: []*Vip{
							{
								Name:    "vip-a",
								Address: "20.0.0.1/32",
							},
						},
					},
				},
				new: []*Conduit_PortNat{
					{
						Port:       10,
						TargetPort: 10,
						Protocol:   "TCP",
						Vips: []*Vip{
							{
								Name:    "vip-b",
								Address: "150.0.0.1/32",
							},
						},
					},
				},
			},
			want: []*Conduit_PortNat{},
			want1: []*Conduit_PortNat{
				{
					Port:       10,
					TargetPort: 10,
					Protocol:   "TCP",
					Vips: []*Vip{
						{
							Name:    "vip-b",
							Address: "150.0.0.1/32",
						},
					},
				},
			},
			want2: []*Conduit_PortNat{},
		},
		{
			name: "different port and TargetPort and protocol",
			args: args{
				old: []*Conduit_PortNat{
					{
						Port:       10,
						TargetPort: 10,
						Protocol:   "TCP",
						Vips:       []*Vip{},
					},
					{
						Port:       10,
						TargetPort: 11,
						Protocol:   "TCP",
						Vips:       []*Vip{},
					},
					{
						Port:       11,
						TargetPort: 10,
						Protocol:   "TCP",
						Vips:       []*Vip{},
					},
				},
				new: []*Conduit_PortNat{
					{
						Port:       10,
						TargetPort: 10,
						Protocol:   "UDP",
						Vips:       []*Vip{},
					},
				},
			},
			want: []*Conduit_PortNat{
				{
					Port:       10,
					TargetPort: 10,
					Protocol:   "UDP",
					Vips:       []*Vip{},
				},
			},
			want1: []*Conduit_PortNat{},
			want2: []*Conduit_PortNat{
				{
					Port:       10,
					TargetPort: 10,
					Protocol:   "TCP",
					Vips:       []*Vip{},
				},
				{
					Port:       10,
					TargetPort: 11,
					Protocol:   "TCP",
					Vips:       []*Vip{},
				},
				{
					Port:       11,
					TargetPort: 10,
					Protocol:   "TCP",
					Vips:       []*Vip{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := PortNatDiff(tt.args.old, tt.args.new)
			if !conduitPortNatEquals(got, tt.want) {
				t.Errorf("PortNatDiff() got = %v, want %v", got, tt.want)
			}
			if !conduitPortNatEquals(got1, tt.want1) {
				t.Errorf("PortNatDiff() got1 = %v, want %v", got1, tt.want1)
			}
			if !conduitPortNatEquals(got2, tt.want2) {
				t.Errorf("PortNatDiff() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

// The order of the result in the TestPortNatDiff test was not always the same,
// so due to reflect.DeepEqual, the tests were not passing all time.
func conduitPortNatEquals(a []*Conduit_PortNat, b []*Conduit_PortNat) bool {
	aMap := map[string][]*Vip{}
	for _, v := range a {
		aMap[v.GetNatName()] = v.GetVips()
	}
	for _, v := range b {
		vipsA, exists := aMap[v.GetNatName()]
		if !exists {
			return false
		}
		if !reflect.DeepEqual(vipsA, v.GetVips()) {
			return false
		}
		delete(aMap, v.GetNatName())
	}
	return len(aMap) == 0
}
