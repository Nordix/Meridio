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

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

type Gateway struct {
	Name       string
	Address    string
	RemoteASN  uint32
	LocalASN   uint32
	RemotePort uint16
	LocalPort  uint16
	IPFamily   string
	BFD        bool
	Protocol   string
	HoldTime   uint
}

func (gw *Gateway) String() string {
	return fmt.Sprintf("name:\"%v\" address:\"%v\" remoteASN:%v localASN:%v remotePort:%v localPort:%v ipFamily:\"%v\" protocol:\"%v\" bfd:%v holdTime:%v",
		gw.Name, gw.Address, gw.RemoteASN, gw.LocalASN, gw.RemotePort, gw.LocalPort, gw.IPFamily, gw.Protocol, gw.BFD, gw.HoldTime)
}

func NewGateway(gateway *nspAPI.Gateway) *Gateway {
	return &Gateway{
		Name:       gateway.GetName(),
		Address:    gateway.GetAddress(),
		RemoteASN:  gateway.GetRemoteASN(),
		LocalASN:   gateway.GetLocalASN(),
		RemotePort: uint16(gateway.GetRemotePort()),
		LocalPort:  uint16(gateway.GetLocalPort()),
		IPFamily:   gateway.GetIpFamily(),
		BFD:        gateway.GetBfd(),
		Protocol:   gateway.GetProtocol(),
		HoldTime:   uint(gateway.GetHoldTime()),
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
