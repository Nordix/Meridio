package loadbalancer

import "github.com/nordix/meridio/pkg/networking"

type Target struct {
	identifier int
	ips        []string
	fwMarks    []networking.FWMarkRoute
}

func (t *Target) GetIdentifier() int {
	return t.identifier
}

func (t *Target) GetIPs() []string {
	return t.ips
}

func (t *Target) Configure(netUtils networking.Utils) error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
	}
	var err error
	for _, ip := range t.ips {
		var fwMark networking.FWMarkRoute
		fwMark, err = netUtils.NewFWMarkRoute(ip, t.identifier, t.identifier)
		t.fwMarks = append(t.fwMarks, fwMark)
	}
	return err
}

func (t *Target) Delete() error {
	if t.fwMarks == nil {
		t.fwMarks = []networking.FWMarkRoute{}
		return nil
	}
	var err error
	for _, fwMark := range t.fwMarks {
		err = fwMark.Delete()
	}
	t.fwMarks = []networking.FWMarkRoute{}
	return err
}

func NewTarget(identifier int, ips []string) *Target {
	target := &Target{
		identifier: identifier,
		ips:        ips,
		fwMarks:    []networking.FWMarkRoute{},
	}
	return target
}
