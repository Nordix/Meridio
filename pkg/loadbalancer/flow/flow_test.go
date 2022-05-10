/*
  SPDX-License-Identifier: Apache-2.0
  Copyright (c) 2022 Nordix Foundation
*/

package flow

import (
	"testing"
	"reflect"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
)

// Load-balancer handler
type lb struct {
	deleteFlow int
	setFlow int
}
var activeLb lb
func (lb *lb) DeleteFlow(flow *nspAPI.Flow) error {
	lb.deleteFlow++
	return nil
}
func (lb *lb) SetFlow(flow *nspAPI.Flow) error {
	lb.setFlow++
	return nil
}
func (*lb) Activate(identifier int) error { return nil }
func (*lb) Deactivate(identifier int) error { return nil }
func (*lb) Start() error { return nil }
func (*lb) Delete() error { return nil }

// NFT handler
type testNfth struct {
	portNATSet int
	portNATDelete int
	dport uint
	localPort uint
	createSets int
	deleteSets int
}
var activeNfth testNfth
func (nfth *testNfth) PortNATSet(flowName string, protocols []string, dport, localPort uint) error {
	nfth.portNATSet++
	nfth.dport = dport
	nfth.localPort = localPort
	return nil
}
func (nfth *testNfth) PortNATDelete(flowName string) {
	nfth.portNATDelete++
	nfth.dport = 0
	nfth.localPort = 0
}
func (nfth *testNfth) PortNATCreateSets(flow *nspAPI.Flow) error {
	nfth.createSets++
	return nil
}
func (nfth *testNfth) PortNATDeleteSets(flow *nspAPI.Flow) {
	nfth.deleteSets++
}
func (nfth *testNfth) PortNATSetAddresses(flow *nspAPI.Flow) error {
	return nil
}



func TestFlow(t *testing.T) {
	var tests = []struct {
		name string
		flowSpec *nspAPI.Flow
		lb types.NFQueueLoadBalancer
		nfth types.NftHandler
		expectError bool
		create bool
	}{
		{
			name: "Create",
			create: true,
			flowSpec: &nspAPI.Flow{Name: "flow1"},
			lb: &lb{0,1},
		},
		{
			name: "Update to same",
			flowSpec: &nspAPI.Flow{Name: "flow1"},
			lb: &lb{0,1},
		},
		{
			name: "Update name",
			flowSpec: &nspAPI.Flow{Name: "flow2"},
			lb: &lb{0,1},
			expectError: true,
		},
		{
			name: "Delete",
			lb: &lb{1,1},
			nfth: &testNfth{0,0,0,0,0,0},
		},
		{ // #4
			name: "Create port-NAT, no dport",
			create: true,
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				LocalPort: 8080,
			},
			expectError: true,
			lb: &lb{0,0},
			nfth: &testNfth{0,0,0,0,0,0},
		},
		{
			name: "Create port-NAT, multi dport",
			create: true,
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"10","20"},
				LocalPort: 8080,
			},
			expectError: true,
			lb: &lb{0,0},
			nfth: &testNfth{0,0,0,0,0,0},
		},
		{ // #6
			name: "Create port-NAT, dport range",
			create: true,
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"10-20"},
				LocalPort: 8080,
			},
			expectError: true,
			lb: &lb{0,0},
			nfth: &testNfth{0,0,0,0,0,0},
		},
		{
			name: "Create port-NAT",
			create: true,
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"80"},
				LocalPort: 8080,
			},
			lb: &lb{0,1},
			nfth: &testNfth{1,0,80,8080,1,0},
		},
		{ // #8
			name: "Update port-NAT, LocalPort",
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"80"},
				LocalPort: 7777,
			},
			lb: &lb{0,2},
			nfth: &testNfth{2,0,80,7777,1,0},
		},
		{
			name: "Update port-NAT, dport",
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"23"},
				LocalPort: 7777,
			},
			lb: &lb{0,3},
			nfth: &testNfth{3,0,23,7777,1,0},
		},
		{ // 10
			name: "Update port-NAT, remove",
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"23"},
			},
			lb: &lb{0,4},
			nfth: &testNfth{3,1,0,0,1,1},
		},
		{
			name: "Re-add port-NAT, dport",
			flowSpec: &nspAPI.Flow{
				Name: "flow-port-NAT",
				DestinationPortRanges: []string{"23"},
				LocalPort: 7777,
			},
			lb: &lb{0,5},
			nfth: &testNfth{4,1,23,7777,2,1},
		},
		{
			name: "Delete",
			lb: &lb{1,5},
			nfth: &testNfth{4,2,0,0,2,2},
		},
		{ // 13
			name: "Create flow without port-NAT",
			create: true,
			flowSpec: &nspAPI.Flow{
				Name: "flow-NO-port-NAT",
				DestinationPortRanges: []string{"80"},
			},
			lb: &lb{0,1},
			nfth: &testNfth{0,0,0,0,0,0},
		},
		{
			name: "Update no-NAT flow to port-NAT",
			flowSpec: &nspAPI.Flow{
				Name: "flow-NO-port-NAT",
				DestinationPortRanges: []string{"80"},
				LocalPort: 7777,
			},
			lb: &lb{0,2},
			nfth: &testNfth{1,0,80,7777,1,0},
		},
		{
			name: "Delete flow-NO-port-NAT",
			lb: &lb{1,2},
			nfth: &testNfth{1,1,0,0,1,1},
		},
	}

	var F types.Flow
	var err error

	for i, tc := range tests {
		if tc.create {
			activeLb = lb{0,0}
			activeNfth = testNfth{0,0,0,0,0,0}
			F, err = New(tc.flowSpec, &activeLb, &activeNfth)
		} else {
			if tc.flowSpec != nil {
				err = F.Update(tc.flowSpec)
			} else {
				err = F.Delete()
				F = nil
			}
		}
		if (tc.expectError) {
			if err == nil {
				t.Fatalf("%s(%d): Expected error but got nil", tc.name, i)
			}
		} else {
			if err != nil {
				t.Fatalf("%s(%d): Unexpected error: %v", tc.name, i, err)
			}
		}
		if tc.lb != nil && !reflect.DeepEqual(tc.lb, &activeLb) {
			t.Fatalf("%s(%d): Lb; expected [%v], got [%v]", tc.name, i, tc.lb, &activeLb)
		}
		if tc.nfth != nil && !reflect.DeepEqual(tc.nfth, &activeNfth) {
			t.Fatalf("%s(%d): Nfth; expected [%v], got [%v]", tc.name, i, tc.nfth, &activeNfth)
		}
	}
}
