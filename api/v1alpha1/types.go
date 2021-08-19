package v1alpha1

import "errors"

type Status struct {
	NoPhase string
	Error   string
}

var BaseStatus = Status{
	NoPhase: "",
	Error:   "error",
}

type ConfigStatusType struct {
	Rejected   string
	Accepted   string
	Disengaged string
	Engaged    string
}

var ConfigStatus = ConfigStatusType{
	Rejected:   "rejected",
	Accepted:   "accepted",
	Disengaged: "disengaged",
	Engaged:    "engaged",
}

type DeploymentStatusType struct {
	Deployed string
}

var DeploymentStatus = DeploymentStatusType{
	Deployed: "deployed",
}

type Protocol string

var BFD Protocol = "static"
var BGP Protocol = "bgp"

type ipFamily string

var (
	IPv4      ipFamily = "ipv4"
	IPv6      ipFamily = "ipv6"
	Dualstack ipFamily = "dualstack"
)

func (f ipFamily) IsValid() error {
	switch f {
	case IPv4, IPv6, Dualstack:
		return nil
	default:
		return errors.New("invalid ip-family type")
	}
}
