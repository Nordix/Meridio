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

// IsValid returns true if the receiver is a valid ipFamily type
func (p Protocol) IsValid() bool {
	switch p {
	case BGP, Static:
		return true
	default:
		return false
	}
}

// +kubebuilder:validation:Enum=ipv4;ipv6;dualstack

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

// IsValid returns true if the receiver is a valid ipFamily type
func (f IPFamily) IsValid() bool {
	switch f {
	case IPv4, IPv6, Dualstack:
		return true
	default:
		return false
	}
}

type TransportProtocol string

const (
	TCP TransportProtocol = "tcp"
	UDP TransportProtocol = "udp"
)

// IsValid returns true if the receiver is a valid TransportProtocol type
func (p TransportProtocol) IsValid() bool {
	switch p {
	case TCP, UDP:
		return true
	default:
		return false
	}
}

type NetworkServiceType string

const (
	StatelessLB = "stateless-lb"
)

// IsValid returns true if the receiver is a valid network service type
func (t NetworkServiceType) IsValid() bool {
	switch t {
	case StatelessLB:
		return true
	default:
		return false
	}
}

func validatePrefix(p string) (*net.IPNet, error) {
	_, n, err := net.ParseCIDR(p)
	if err != nil {
		return nil, err
	}
	if n.String() != p {
		return nil, fmt.Errorf("%s is not a valid prefix, probably %v should be used", p, n)
	}
	return n, nil
}

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
	formaterr := fmt.Errorf("port %s is invalid, valid format should be either a port range or a single port, example:35000-35500 or 40000", p)
	var ports []string
	if strings.Contains(p, "-") {
		ports = strings.Split(p, "-")
		if len(ports) != 2 {
			return Ports{}, formaterr
		}
	} else {
		ports = []string{p, p}
	}
	var portsUint []uint64
	for _, p := range ports {
		s, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return Ports{}, formaterr
		}
		portsUint = append(portsUint, s)
	}
	if portsUint[0] > portsUint[1] {
		return Ports{}, formaterr
	}
	return Ports{portsUint[0], portsUint[1]}, nil
}
