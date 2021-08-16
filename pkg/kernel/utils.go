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
	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
)

type KernelUtils struct {
}

func (ku *KernelUtils) NewInterface(index int) networking.Iface {
	return NewInterface(index)
}

func (ku *KernelUtils) NewBridge(name string) (networking.Bridge, error) {
	return NewBridge(name)
}

func (ku *KernelUtils) NewFWMarkRoute(ip string, fwmark int, tableID int) (networking.FWMarkRoute, error) {
	return NewFWMarkRoute(ip, fwmark, tableID)
}

func (ku *KernelUtils) NewNFQueue(prefix string, queueNum int) (networking.NFQueue, error) {
	return NewNFQueue(prefix, queueNum)
}

func (ku *KernelUtils) NewSourceBasedRoute(tableID int, prefix string) (networking.SourceBasedRoute, error) {
	return NewSourceBasedRoute(tableID, prefix)
}

func (ku *KernelUtils) NewInterfaceMonitor() (networking.InterfaceMonitor, error) {
	return NewInterfaceMonitor()
}

// func (ku *KernelUtils) IfaceByName(name string) (networking.Iface, error) {

// }

func (ku *KernelUtils) GetIndexFromName(name string) (int, error) {
	return GetIndexFromName(name)
}

func (ku *KernelUtils) AddVIP(vip string) error {
	return AddVIP(vip)
}

func (ku *KernelUtils) DeleteVIP(vip string) error {
	return DeleteVIP(vip)
}

func getLink(intf networking.Iface) (netlink.Link, error) {
	return netlink.LinkByIndex(intf.GetIndex())
}

// GetIndexFromName - Get the index of an interface from its name
func GetIndexFromName(name string) (int, error) {
	intf, err := netlink.LinkByName(name)
	if err != nil {
		return -1, err
	}
	return intf.Attrs().Index, nil
}
