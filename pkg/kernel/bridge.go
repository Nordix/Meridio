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
	"fmt"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
)

type Bridge struct {
	name             string
	linkedInterfaces []networking.Iface
	Interface
}

func (b *Bridge) create() error {
	mac, err := networking.GenerateMacAddress()
	if err != nil {
		return fmt.Errorf("failed to generate mac address (bridge): %w", err)
	}
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:         b.name,
			HardwareAddr: mac,
		},
	}
	err = netlink.LinkAdd(bridge)
	if err != nil {
		return fmt.Errorf("failed to LinkAdd (bridge): %w", err)
	}
	err = netlink.LinkSetUp(bridge)
	if err != nil {
		return fmt.Errorf("failed to LinkSetUp (bridge): %w", err)
	}
	b.index = bridge.Index
	return nil
}

func (b *Bridge) useExistingBridge() error {
	index, err := GetIndexFromName(b.name)
	if err != nil {
		return err
	}
	b.index = index
	return nil
}

func (b *Bridge) Delete() error {
	return nil
}

func (b *Bridge) addTolinkedInterfaces(intf networking.Iface) {
	b.linkedInterfaces = append(b.linkedInterfaces, intf)
}

func (b *Bridge) removeFromlinkedInterfaces(intf networking.Iface) {
	for index, i := range b.linkedInterfaces {
		if i.Equals(intf) {
			b.linkedInterfaces = append(b.linkedInterfaces[:index], b.linkedInterfaces[index+1:]...)
		}
	}
}

func (b *Bridge) interfaceIsLinked(intf networking.Iface) bool {
	for _, i := range b.linkedInterfaces {
		if i.Equals(intf) {
			return true
		}
	}
	return false
}

// LinkInterface set the bridge as master of another interface
func (b *Bridge) LinkInterface(intf networking.Iface) error {
	if b.interfaceIsLinked(intf) {
		return nil // TODO
	}
	b.addTolinkedInterfaces(intf)
	bridgeLink, err := b.getLink()
	if err != nil {
		return err
	}
	interfaceLink, err := getLink(intf)
	if err != nil {
		return err
	}
	err = netlink.LinkSetMaster(interfaceLink, bridgeLink)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bridge) UnLinkInterface(intf networking.Iface) error {
	if !b.interfaceIsLinked(intf) {
		return nil // TODO
	}
	b.removeFromlinkedInterfaces(intf)
	interfaceLink, err := getLink(intf)
	if err != nil {
		return err
	}
	return netlink.LinkSetNoMaster(interfaceLink)
}

func NewBridge(name string) (*Bridge, error) {
	bridge := &Bridge{
		name:             name,
		linkedInterfaces: []networking.Iface{},
		Interface:        Interface{},
	}
	err := bridge.create()
	if err != nil {
		err = bridge.useExistingBridge()
		if err != nil {
			return nil, err
		}
	}
	return bridge, nil
}
