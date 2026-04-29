/*
Copyright (c) 2026 OpenInfra Foundation Europe

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
	"testing"
)

func uint16Ptr(v uint16) *uint16 { return &v }
func uint32Ptr(v uint32) *uint32 { return &v }
func boolPtr(v bool) *bool       { return &v }

func TestValidateSpec_StaticWithBFD(t *testing.T) {
	gw := &Gateway{
		Spec: GatewaySpec{
			Address:  "169.254.100.150",
			Protocol: "static",
			Static: StaticSpec{
				BFD: BfdSpec{
					Switch:     boolPtr(true),
					MinTx:      "300ms",
					MinRx:      "300ms",
					Multiplier: uint16Ptr(3),
				},
			},
		},
	}

	errs := gw.validateSpec()
	if errs != nil {
		t.Errorf("expected no errors for static+BFD, got: %v", errs)
	}
}

func TestValidateSpec_StaticWithAutoDefaultedBGPPorts(t *testing.T) {
	// Simulates what happens when the API server applies CRD defaults
	// to the bgp section during an Update (remote-port and local-port get set to 179)
	gw := &Gateway{
		Spec: GatewaySpec{
			Address:  "169.254.100.150",
			Protocol: "static",
			Static: StaticSpec{
				BFD: BfdSpec{
					Switch:     boolPtr(true),
					MinTx:      "300ms",
					MinRx:      "300ms",
					Multiplier: uint16Ptr(3),
				},
			},
			Bgp: BgpSpec{
				RemotePort: uint16Ptr(179),
				LocalPort:  uint16Ptr(179),
			},
		},
	}

	errs := gw.validateSpec()
	if errs != nil {
		t.Errorf("expected no errors for static with auto-defaulted BGP ports, got: %v", errs)
	}
}

func TestValidateSpec_StaticWithBGPFields_Rejected(t *testing.T) {
	gw := &Gateway{
		Spec: GatewaySpec{
			Address:  "169.254.100.150",
			Protocol: "static",
			Bgp: BgpSpec{
				RemoteASN: uint32Ptr(1234),
				LocalASN:  uint32Ptr(4321),
			},
		},
	}

	errs := gw.validateSpec()
	if errs == nil {
		t.Error("expected validation error for static with BGP ASN fields set")
	}
}
