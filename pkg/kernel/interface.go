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

package kernel

import (
	"reflect"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
)

type Interface struct {
	index         int
	LocalIPs      []string
	NeighborIPs   []string
	Gateways      []string
	InterfaceType networking.InterfaceType
}

func (intf *Interface) getLink() (netlink.Link, error) {
	return netlink.LinkByIndex(intf.index)
}

func (intf *Interface) GetIndex() int {
	return intf.index
}

func (intf *Interface) GetName() string {
	i, err := intf.getLink()
	if err != nil {
		return ""
	}
	return i.Attrs().Name
}

func (intf *Interface) GetLocalPrefixes() []string {
	return intf.LocalIPs
}

func (intf *Interface) SetLocalPrefixes(localPrefixes []string) {
	intf.LocalIPs = localPrefixes
}

func (intf *Interface) GetNeighborPrefixes() []string {
	return intf.NeighborIPs
}

func (intf *Interface) SetNeighborPrefixes(neighborPrefixes []string) {
	intf.NeighborIPs = neighborPrefixes
}

func (intf *Interface) GetGatewayPrefixes() []string {
	return intf.Gateways
}

func (intf *Interface) SetGatewayPrefixes(gateways []string) {
	intf.Gateways = gateways
}

func (intf *Interface) GetInterfaceType() networking.InterfaceType {
	return intf.InterfaceType
}

func (intf *Interface) SetInterfaceType(ifaceType networking.InterfaceType) {
	intf.InterfaceType = ifaceType
}

func (intf *Interface) AddLocalPrefix(prefix string) error {
	addr, err := netlink.ParseAddr(prefix)
	if err != nil {
		return err
	}
	addr.Label = ""
	i, err := intf.getLink()
	if err != nil {
		return err
	}
	err = netlink.AddrAdd(i, addr)
	if err != nil {
		return err
	}
	return nil
}

func (intf *Interface) RemoveLocalPrefix(prefix string) error {
	addr, err := netlink.ParseAddr(prefix)
	if err != nil {
		return err
	}
	i, err := intf.getLink()
	if err != nil {
		return err
	}
	err = netlink.AddrDel(i, addr)
	if err != nil {
		return err
	}
	return nil
}

func (intf *Interface) Equals(iface networking.Iface) bool {
	return reflect.DeepEqual(intf, iface)
}

func NewInterface(index int) *Interface {
	intf := &Interface{
		index:         index,
		LocalIPs:      []string{},
		NeighborIPs:   []string{},
		Gateways:      []string{},
		InterfaceType: -1,
	}
	return intf
}
