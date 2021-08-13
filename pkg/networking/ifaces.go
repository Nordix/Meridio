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

package networking

const (
	NSE = iota // Interface linked to a NSC (e.g. target)
	NSC        // Interface linked to a NSE (e.g. Load balancer)
)

type InterfaceType int

type Iface interface {
	GetIndex() int
	GetName() string

	GetLocalPrefixes() []string
	SetLocalPrefixes(localPrefixes []string)
	AddLocalPrefix(prefix string) error
	RemoveLocalPrefix(prefix string) error

	GetNeighborPrefixes() []string
	SetNeighborPrefixes(neighborPrefixes []string)

	GetGatewayPrefixes() []string
	SetGatewayPrefixes(gateways []string)

	GetInterfaceType() InterfaceType
	SetInterfaceType(ifaceType InterfaceType)

	Equals(Iface) bool
}

type Utils interface {
	NewInterface(index int) Iface
	NewBridge(name string) (Bridge, error)
	NewFWMarkRoute(ip string, fwmark int, tableID int) (FWMarkRoute, error)
	NewNFQueue(prefix string, queueNum int) (NFQueue, error)
	NewSourceBasedRoute(tableID int, prefix string) (SourceBasedRoute, error)

	NewInterfaceMonitor() (InterfaceMonitor, error)

	GetIndexFromName(name string) (int, error)
	AddVIP(vip string) error
	DeleteVIP(vip string) error
}

type Bridge interface {
	Iface
	LinkInterface(intf Iface) error
	UnLinkInterface(intf Iface) error
}

type FWMarkRoute interface {
	Delete() error
}

type NFQueue interface {
	Delete() error
}

type SourceBasedRoute interface {
	Delete() error
	AddNexthop(nexthop string) error
	RemoveNexthop(nexthop string) error
}

type InterfaceMonitor interface {
	Subscribe(subscriber InterfaceMonitorSubscriber)
	UnSubscribe(subscriber InterfaceMonitorSubscriber)
	Close()
}

type InterfaceMonitorSubscriber interface {
	InterfaceCreated(Iface)
	InterfaceDeleted(Iface)
}
