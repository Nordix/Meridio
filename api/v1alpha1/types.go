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

var BFD Protocol = "static"
var BGP Protocol = "bgp"
