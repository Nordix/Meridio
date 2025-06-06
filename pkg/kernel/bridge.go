/*
Copyright (c) 2021-2023 Nordix Foundation
Copyright (c) 2025 OpenInfra Foundation Europe

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

	"github.com/nordix/meridio/pkg/log"
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
	link, err := netlink.LinkByName(b.name)
	if err != nil {
		return fmt.Errorf("failed getting bridge by name (%s): %w", b.name, err)
	}

	index := link.Attrs().Index
	// make sure the link is up
	err = netlink.LinkSetUp(link)
	if err != nil {
		return fmt.Errorf("failed to LinkSetUp existing bridge: %w", err)
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

func (b *Bridge) InterfaceIsLinked(intf networking.Iface) bool {
	for _, i := range b.linkedInterfaces {
		if i.Equals(intf) {
			return true
		}
	}
	return false
}

// FindLinkedInterfaceByIndex -
// Finds and returns a linked interface by index.
func (b *Bridge) FindLinkedInterfaceByIndex(index int) networking.Iface {
	if index < 0 {
		return nil
	}
	for _, i := range b.linkedInterfaces {
		if i.GetIndex() == index {
			return i
		}
	}
	return nil
}

// LinkInterface sets the bridge as master of another interface
//
// Note: InterfaceIsLinked() relies on DeepEqual, thus the same
// interface might be added multiple times to linkedInterfaces.
// Therefore, do a lookup based on the interface index, and overwrite
// existing linked interface with the new if interface type matches.
// TODO: Check if interface type update is possible.
func (b *Bridge) LinkInterface(intf networking.Iface) error {
	if b.InterfaceIsLinked(intf) {
		return nil
	}
	// check for linked interface with the same index, and replace it
	if linked := b.FindLinkedInterfaceByIndex(intf.GetIndex()); linked != nil {
		log.Logger.Info("Interface with index already linked to bridge",
			"index", intf.GetIndex(), "linked interface", linked, "new interface", intf)
		// Note: Type update propably should not happen, but I don't know the impacts if it would be possible.
		if linked.GetInterfaceType() == intf.GetInterfaceType() {
			b.removeFromlinkedInterfaces(linked)
		}
	}
	// link the interface
	b.addTolinkedInterfaces(intf)
	bridgeLink, err := b.getLink()
	if err != nil {
		return fmt.Errorf("failed to getLink bridge while linking interface (%s): %w", b.GetName(), err)
	}
	interfaceLink, err := getLink(intf)
	if err != nil {
		return fmt.Errorf("failed to getLink interface while linking interface (%s): %w",
			intf.GetName(networking.WithNoLoad()), err)
	}
	err = netlink.LinkSetMaster(interfaceLink, bridgeLink)
	if err != nil {
		return fmt.Errorf("failed to LinkSetMaster while linking interface (%s - %s): %w",
			b.GetName(), intf.GetName(networking.WithNoLoad()), err)
	}
	return nil
}

func (b *Bridge) UnLinkInterface(intf networking.Iface) error {
	if !b.InterfaceIsLinked(intf) {
		return nil
	}
	b.removeFromlinkedInterfaces(intf)
	interfaceLink, err := getLink(intf)
	if err != nil {
		return fmt.Errorf("failed to getLink interface while unlinking interface (%s): %w",
			intf.GetName(networking.WithNoLoad()), err)
	}

	err = netlink.LinkSetNoMaster(interfaceLink)
	if err != nil {
		return fmt.Errorf("failed to LinkSetNoMaster interface while unlinking interface (%s): %w",
			intf.GetName(networking.WithNoLoad()), err)
	}

	return nil
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
