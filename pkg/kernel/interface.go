/*
Copyright (c) 2021-2023 Nordix Foundation

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
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
)

type Interface struct {
	index         int
	Name          string
	LocalIPs      []string
	NeighborIPs   []string
	Gateways      []string
	InterfaceType networking.InterfaceType
}

func (intf *Interface) getLink() (netlink.Link, error) {
	link, err := netlink.LinkByIndex(intf.index)
	if err != nil {
		return link, fmt.Errorf("failed LinkByIndex (%d): %w", intf.index, err)
	}
	return link, nil
}

func (intf *Interface) GetIndex() int {
	return intf.index
}

func (intf *Interface) GetName(options ...networking.IfaceNameOption) string {
	opts := &networking.IfaceNameOptions{
		NoResolve: false,
		NoLoad:    false,
	}
	for _, option := range options {
		option(opts)
	}
	// try to resolve name by interface index via netlink
	if !opts.NoResolve && intf.Name == "" {
		i, err := intf.getLink()
		if err != nil {
			return ""
		}
		if opts.NoLoad {
			// return name without changing the interface struct
			return i.Attrs().Name
		}
		intf.Name = i.Attrs().Name
	}
	return intf.Name
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
		return fmt.Errorf("failed ParseAddr while adding local prefix (%s): %w", prefix, err)
	}
	addr.Label = ""
	i, err := intf.getLink()
	if err != nil {
		return fmt.Errorf("failed getLink (%s) while adding local prefix (%s): %w", intf.GetName(), prefix, err)
	}
	err = netlink.AddrAdd(i, addr)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed AddrAdd (%s) while adding local prefix (%s): %w", intf.GetName(), prefix, err)
	}
	return nil
}

func (intf *Interface) RemoveLocalPrefix(prefix string) error {
	addr, err := netlink.ParseAddr(prefix)
	if err != nil {
		return fmt.Errorf("failed ParseAddr while removing local prefix (%s): %w", prefix, err)
	}
	i, err := intf.getLink()
	if err != nil {
		return fmt.Errorf("failed getLink (%s) while removing local prefix (%s): %w", intf.GetName(), prefix, err)
	}
	err = netlink.AddrDel(i, addr)
	if err != nil {
		return fmt.Errorf("failed AddrDel (%s) while removing local prefix (%s): %w", intf.GetName(), prefix, err)
	}
	return nil
}

func (intf *Interface) Equals(iface networking.Iface) bool {
	return intf.GetIndex() == iface.GetIndex() &&
		intf.GetInterfaceType() == iface.GetInterfaceType() &&
		intf.GetName(networking.WithNoResolve()) == iface.GetName(networking.WithNoResolve()) &&
		equalStringList(intf.GetLocalPrefixes(), iface.GetLocalPrefixes()) &&
		equalStringList(intf.GetNeighborPrefixes(), iface.GetNeighborPrefixes()) &&
		equalStringList(intf.GetGatewayPrefixes(), iface.GetGatewayPrefixes())

}

func equalStringList(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	tmpa := make([]string, len(a))
	tmpb := make([]string, len(b))
	copy(tmpa, a)
	copy(tmpb, b)
	sort.Strings(tmpa)
	sort.Strings(tmpb)

	for i := range tmpa {
		if tmpa[i] != tmpb[i] {
			return false
		}
	}
	return true
}

func NewInterface(index int, options ...InterfaceOption) *Interface {
	opts := &interfaceOptions{}
	for _, opt := range options {
		opt(opts)
	}
	intf := &Interface{
		index:         index,
		Name:          opts.name,
		LocalIPs:      []string{},
		NeighborIPs:   []string{},
		Gateways:      []string{},
		InterfaceType: -1,
	}
	return intf
}

type InterfaceOption func(o *interfaceOptions)

type interfaceOptions struct {
	name string
}

func WithInterfaceName(name string) InterfaceOption {
	return func(o *interfaceOptions) {
		o.name = name
	}
}
