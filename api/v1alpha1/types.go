/*
Copyright (c) 2021-2022 Nordix Foundation

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

package v1alpha1

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ConfigStatus describes the status of a meridio operator resource to indicate if the resource is ready to use or not
type ConfigStatus string

const (
	// Normally when a resouce is not processed by the corresponding controller, the status will be NoStatus
	NoPhase ConfigStatus = ""

	// If the validation of a resource does not pass in the controller, the status will be Error
	Error ConfigStatus = "error"

	// Normally when a resource is not created in a correct sequence, the status will be Disengaged
	Disengaged ConfigStatus = "disengaged"

	// Engaged indicates the resouce is readly to be used.
	Engaged ConfigStatus = "engaged"
)

// Protocol describes the routing choice of the frontend
type Protocol string

const (
	// Static instructs the frontend to work with the static routing configured on the Edge Routers
	Static Protocol = "static"

	// BGP instructs the frontend to setup BGP sessions with the Edge Routers
	BGP Protocol = "bgp"
)

// IPFamily describes the traffic type in the trench
// Only one of the following ip family can be specified.
// If the traffic is IPv4 only, use IPv4, similarly,
// use IPv6 if the traffic is IPv6 only, otherwise, use
// dualstack which handles both IPv4 and IPv6 traffic.
type IPFamily string

const (
	IPv4      IPFamily = "ipv4"
	IPv6      IPFamily = "ipv6"
	Dualstack IPFamily = "dualstack"
)

// +kubebuilder:validation:Enum=tcp;udp;sctp
type TransportProtocol string

const (
	TCP  TransportProtocol = "tcp"
	UDP  TransportProtocol = "udp"
	SCTP TransportProtocol = "sctp"
)

func TransportProtocolsToStrings(proto []TransportProtocol) []string {
	var ret []string
	for _, p := range proto {
		ret = append(ret, string(p))
	}
	return ret
}

type NetworkServiceType string

const (
	StatelessLB = "stateless-lb"
)

func validatePrefix(p string) (*net.IPNet, error) {
	ip, n, err := net.ParseCIDR(p)
	if err != nil {
		return nil, err
	}
	if !ip.Equal(n.IP) {
		return nil, fmt.Errorf("%s is not a valid prefix, probably %v should be used", p, n)
	}
	return n, nil
}

type InterfaceType string

const (
	NSMVlan = "nsm-vlan"
)

func subnetsOverlap(a, b *net.IPNet) bool {
	return subnetContainsSubnet(a, b) || subnetContainsSubnet(b, a)
}

func subnetContainsSubnet(outer, inner *net.IPNet) bool {
	ol, _ := outer.Mask.Size()
	il, _ := inner.Mask.Size()
	if ol == il && outer.IP.Equal(inner.IP) {
		return true
	}
	if ol < il && outer.Contains(inner.IP) {
		return true
	}
	return false
}

type Ports struct {
	Start uint64
	End   uint64
}

func validPortsFormat(p string) (Ports, error) {
	var ports []string
	if strings.Contains(p, "-") {
		ports = strings.Split(p, "-")
		if len(ports) != 2 {
			return Ports{}, fmt.Errorf("wrong format to define port range, <starting port>-<ending port>")
		}
		return NewPortFromString(ports[0], ports[1])
	} else if p == "any" {
		return NewPort(0, 65535)
	} else {
		return NewPortFromString(p, p)
	}
}

func NewPortFromString(start, end string) (Ports, error) {
	startInt, err := strconv.ParseUint(start, 10, 16)
	if err != nil {
		return Ports{}, fmt.Errorf("starting port %s is not a valid port number, an integer between 0 and 65535", start)
	}
	endInt, err := strconv.ParseUint(end, 10, 16)
	if err != nil {
		return Ports{}, fmt.Errorf("ending port %s is not a valid port number, an integer between 0 and 65535", end)
	}
	return NewPort(startInt, endInt)
}

func NewPort(start, end uint64) (Ports, error) {
	if start > end {
		return Ports{}, fmt.Errorf("starting port cannot be larger than ending port in the ports range")
	}
	return Ports{start, end}, nil
}
