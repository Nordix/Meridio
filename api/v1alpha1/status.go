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
	Rejected string
	Accepted string
}

var ConfigStatus = ConfigStatusType{
	Rejected: "rejected",
	Accepted: "accepted",
}

type LBStatusType struct {
	Disengaged string
	Engaged    string
}

var LBStatus = LBStatusType{
	Disengaged: "disengaged",
	Engaged:    "engaged",
}

type DeploymentStatusType struct {
	Deployed string
}

var DeploymentStatus = DeploymentStatusType{
	Deployed: "deployed",
}
