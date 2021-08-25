package v1alpha1

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
	case BGP:
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
