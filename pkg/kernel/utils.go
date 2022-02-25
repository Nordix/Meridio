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
	"context"

	"github.com/nordix/meridio/pkg/networking"
	"github.com/vishvananda/netlink"
)

type contextKeyType string

const interfaceMonitorKey contextKeyType = "interfaceMonitor"

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

func (ku *KernelUtils) NewNFQueue(name string, nfqueueNumber uint16, protocols []string, sourceIPs []string, destinationIPs []string, sourcePorts []string, destinationPorts []string, priority int32) (networking.NFQueue, error) {
	return NewNFQueue(name, nfqueueNumber, protocols, sourceIPs, destinationIPs, sourcePorts, destinationPorts, priority)
}

func (ku *KernelUtils) NewSourceBasedRoute(tableID int, prefix string) (networking.SourceBasedRoute, error) {
	return NewSourceBasedRoute(tableID, prefix)
}

func (ku *KernelUtils) NewInterfaceMonitor() (networking.InterfaceMonitor, error) {
	return NewInterfaceMonitor()
}

// WithInterfaceMonitor -
// Stores InterfaceMonitor in Context
func (ku *KernelUtils) WithInterfaceMonitor(parent context.Context, monitor networking.InterfaceMonitor) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, interfaceMonitorKey, monitor)
}

// GetInterfaceMonitor -
// Returns InterfaceMonitor from Context
func (ku *KernelUtils) GetInterfaceMonitor(ctx context.Context) networking.InterfaceMonitor {
	rv, ok := ctx.Value(interfaceMonitorKey).(networking.InterfaceMonitor)
	if ok && rv != nil {
		return rv
	}
	return nil
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
