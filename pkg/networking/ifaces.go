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

package networking

import "context"

const (
	NSE = iota // Interface linked to a NSC (e.g. target)
	NSC        // Interface linked to a NSE (e.g. Load balancer)
)

type InterfaceType int

type IfaceNameOptions struct {
	NoResolve bool
	NoLoad    bool
}

type IfaceNameOption func(*IfaceNameOptions)

// WithNoResolve -
// WithNoResolve implements option pattern for GetName function.
// Tells not to resolve the interface name if not known.
func WithNoResolve() IfaceNameOption {
	return func(in *IfaceNameOptions) {
		in.NoResolve = true
	}
}

// WithNoLoad -
// WithNoLoad implements option pattern for GetName function.
// Tells not to load resolved interface name into the Name field of Interface struct.
func WithNoLoad() IfaceNameOption {
	return func(in *IfaceNameOptions) {
		in.NoLoad = true
	}
}

type Iface interface {
	GetIndex() int
	GetName(options ...IfaceNameOption) string

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
	NewSourceBasedRoute(tableID int, prefix string) (SourceBasedRoute, error)

	NewInterfaceMonitor() (InterfaceMonitor, error)
	WithInterfaceMonitor(parent context.Context, monitor InterfaceMonitor) context.Context
	GetInterfaceMonitor(ctx context.Context) InterfaceMonitor

	GetIndexFromName(name string) (int, error)
}

type Bridge interface {
	Iface
	LinkInterface(intf Iface) error
	UnLinkInterface(intf Iface) error
	InterfaceIsLinked(intf Iface) bool
	FindLinkedInterfaceByIndex(index int) Iface
}

type FWMarkRoute interface {
	Verify() bool
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
