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

package utils

import (
	"fmt"
	"reflect"
	"strings"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type BfdSpec struct {
	MinTx      uint32
	MinRx      uint32
	Multiplier uint32
}

func (b *BfdSpec) String() string {
	return fmt.Sprintf("BfdSpec:{MinTx:%v MinRx:%v Multiplier:%v}",
		b.MinTx, b.MinRx, b.Multiplier)
}

func NewBfdSpec(spec *nspAPI.Gateway_BfdSpec) *BfdSpec {
	if spec == nil {
		return nil
	}
	return &BfdSpec{
		MinTx:      spec.GetMinTx(),
		MinRx:      spec.GetMinRx(),
		Multiplier: spec.GetMultiplier(),
	}
}

type neighbor struct {
	ip string // IP string format without subnet
	af int    // AF family
}

type Gateway struct {
	Name       string
	Address    string
	IPFamily   string
	Protocol   string
	RemoteASN  uint32
	LocalASN   uint32
	RemotePort uint16
	LocalPort  uint16
	HoldTime   uint
	BFD        bool
	BfdSpec    *BfdSpec
	neighbor   *neighbor
}

func (gw *Gateway) String() string {
	return fmt.Sprintf("name:%v address:%v ipFamily:%v protocol:%v "+
		"remoteASN:%v localASN:%v remotePort:%v localPort:%v holdTime:%v bfd:%v%v",
		gw.Name, gw.Address, gw.IPFamily, gw.Protocol, gw.RemoteASN, gw.LocalASN,
		gw.RemotePort, gw.LocalPort, gw.HoldTime, gw.BFD, func() string {
			if !gw.BFD {
				return ""
			} else {
				return " " + gw.BfdSpec.String()
			}
		}())
}

func (gw *Gateway) GetNeighbor() string {
	return gw.neighbor.ip
}

func (gw *Gateway) GetAF() int {
	return gw.neighbor.af
}

func NewGateway(gateway *nspAPI.Gateway) *Gateway {
	return &Gateway{
		Name:       gateway.GetName(),
		Address:    gateway.GetAddress(),
		IPFamily:   gateway.GetIpFamily(),
		Protocol:   gateway.GetProtocol(),
		RemoteASN:  gateway.GetRemoteASN(),
		LocalASN:   gateway.GetLocalASN(),
		RemotePort: uint16(gateway.GetRemotePort()),
		LocalPort:  uint16(gateway.GetLocalPort()),
		HoldTime:   uint(gateway.GetHoldTime()),
		BFD:        gateway.GetBfd(),
		BfdSpec: func() *BfdSpec {
			if !gateway.GetBfd() {
				return nil
			}
			return NewBfdSpec(gateway.GetBfdSpec())
		}(),
		neighbor: &neighbor{
			ip: strings.Split(gateway.GetAddress(), "/")[0],
			af: GetAF(gateway.GetAddress()),
		},
	}
}

func ConvertGateways(gateways []*nspAPI.Gateway) []*Gateway {
	list := []*Gateway{}
	for _, gateway := range gateways {
		list = append(list, NewGateway(gateway))
	}
	return list
}

func DiffGateways(a, b []*Gateway) bool {
	if len(a) != len(b) {
		// different length
		return true
	}

	mapA := makeMapFromGWList(a)
	mapB := makeMapFromGWList(b)
	return func() bool {
		for name := range mapA {
			if _, ok := mapB[name]; !ok {
				return true
			}
		}
		for name := range mapB {
			if _, ok := mapA[name]; !ok {
				return true
			}
		}
		for key, value := range mapA {
			if !reflect.DeepEqual(mapB[key], value) {
				return true
			}
		}
		return false
	}()
}

func makeMapFromGWList(gateways []*Gateway) map[string]Gateway {
	list := gateways
	ret := make(map[string]Gateway)
	for _, item := range list {
		ret[item.Name] = *item
	}
	return ret
}
