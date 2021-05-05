package loadbalancer

type Target struct {
	identifier int
	ip         string
}

func (t *Target) GetIdentifier() int {
	return t.identifier
}

func (t *Target) GetIP() string {
	return t.ip
}

func NewTarget(identifier int, ip string) *Target {
	target := &Target{
		identifier: identifier,
		ip:         ip,
	}
	return target
}
