package v1alpha1

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

var Static Protocol = "static"
var BGP Protocol = "bgp"

func (p Protocol) IsValid() bool {
	switch p {
	case BGP:
		return true
	default:
		return false
	}
}

type ipFamily string

var (
	IPv4      ipFamily = "ipv4"
	IPv6      ipFamily = "ipv6"
	Dualstack ipFamily = "dualstack"
)

func (f ipFamily) IsValid() bool {
	switch f {
	case IPv4, IPv6, Dualstack:
		return true
	default:
		return false
	}
}
