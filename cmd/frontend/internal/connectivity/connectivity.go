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

package connectivity

import (
	"fmt"
	"syscall"
)

const (
	IPv4Up       = uint64(1 << iota)           // FE has IPv4 external connectivity
	IPv6Up                                     // FE has IPv6 external connectivity
	NoIPv4Config                               // No IPv4 Gateways configured
	NoIPv6Config                               // No IPv6 Gateways configured
	AnyGWDown                                  // Not all configured gateways are available
	Up           = IPv4Up | IPv6Up             // FE has IPv4 and IPv6 external connectivity
	NoConfig     = NoIPv4Config | NoIPv6Config // No Gateways configured at all
)

func NewConnectivityStatus() *ConnectivityStatus {
	return &ConnectivityStatus{
		statusMap: map[string]bool{},
	}
}

// ConnectivityStatus -
// Keeps track a Frontend's external connectivity
type ConnectivityStatus struct {
	status    uint64
	statusMap map[string]bool
	log       string
}

// SetGatewayDown -
// Indicate that a gateway is down
func (cs *ConnectivityStatus) SetGatewayDown(name string) {
	cs.statusMap[name] = false // neighbor protocol down
	cs.status |= AnyGWDown     // configured session not Established; mark it (used by logging)
}

// SetGatewayUp -
// Indicate that a gateway is up
// Note: in case at least 1 configured gateway for an IP version is up,
// then external connectivity for respective IP version is considered up
func (cs *ConnectivityStatus) SetGatewayUp(name string, family int) {
	cs.statusMap[name] = true // neighbor protocol up
	if cs.status&Up != Up {
		if family == syscall.AF_INET {
			cs.status |= IPv4Up // at least 1 configured ipv4 gw up
		} else if family == syscall.AF_INET6 {
			cs.status |= IPv6Up // at least 1 configured ipv6 gw up
		}
	}
}

// NoConnectivity -
// Returns external connectivity status based on the stored information
// Note: IPv4 and IPv6 ext connectivity are not handled separately,
// as Meridio NSM "backplane" currently neither is capable doing so.
// However if one IP version lacks configuration while the other have
// working gateway connectivity, then external connectivity is considered up.
func (cs *ConnectivityStatus) NoConnectivity() bool {
	return cs.status&NoConfig == NoConfig || (cs.status&NoIPv4Config == 0 && cs.status&IPv4Up == 0) || (cs.status&NoIPv6Config == 0 && cs.status&IPv6Up == 0)
}

// SetNoConfig -
// Indicate that IPv4/IPv6 has no configuration
func (cs *ConnectivityStatus) SetNoConfig(family int) {
	if family == syscall.AF_INET {
		cs.status |= NoIPv4Config
	} else if family == syscall.AF_INET6 {
		cs.status |= NoIPv6Config
	}
}

// AnyGatewayDown -
// Returns true if at least 1 gateway is down
func (cs *ConnectivityStatus) AnyGatewayDown() bool {
	return cs.status&AnyGWDown != 0
}

func (cs *ConnectivityStatus) Log() string {
	return cs.log
}

func (cs *ConnectivityStatus) Logp() *string {
	return &cs.log
}

func (cs *ConnectivityStatus) Status() uint64 {
	return cs.status
}

func (cs *ConnectivityStatus) StatusMap() map[string]bool {
	return cs.statusMap
}

func (cs *ConnectivityStatus) String() string {
	return fmt.Sprintf("%v, %v, \n%v", cs.status, cs.statusMap, cs.log)
}
