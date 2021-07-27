package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
)

type Resources interface {
	getAction(e *common.Executor) (common.Action, error)
	getModel() error
}

type Meridio struct {
	ipamDeployment *IpamDeployment
	ipamService    *IpamService
	serviceAccount *ServiceAccount
	role           *Role
	roleBinding    *RoleBinding
	nspDeployment  *NspDeployment
	nspService     *NspService
	proxy          *Proxy
}

func NewMeridio(trench *meridiov1alpha1.Trench) (*Meridio, error) {
	ipamsvc, err := NewIPAMSvc(trench)
	if err != nil {
		return nil, err
	}
	ipam, err := NewIPAM(trench)
	if err != nil {
		return nil, err
	}
	sa, err := NewServiceAccount(trench)
	if err != nil {
		return nil, err
	}
	role, err := NewRole(trench)
	if err != nil {
		return nil, err
	}
	rb, err := NewRoleBinding(trench)
	if err != nil {
		return nil, err
	}
	nspd, err := NewNspDeployment(trench)
	if err != nil {
		return nil, err
	}
	nsps, err := NewNspService(trench)
	if err != nil {
		return nil, err
	}
	p, err := NewProxy(trench)
	if err != nil {
		return nil, err
	}
	return &Meridio{
		ipamDeployment: ipam,
		ipamService:    ipamsvc,
		serviceAccount: sa,
		role:           role,
		roleBinding:    rb,
		nspDeployment:  nspd,
		nspService:     nsps,
		proxy:          p,
	}, nil
}

func (m Meridio) ReconcileAll(e *common.Executor, cr *meridiov1alpha1.Trench) error {
	var actions []common.Action
	resources := []Resources{
		m.ipamService,
		m.serviceAccount,
		m.role,
		m.roleBinding,
		m.nspDeployment,
		m.nspService,
		m.proxy,
		m.ipamDeployment,
	}

	for _, r := range resources {
		action, err := r.getAction(e)
		if err != nil {
			return fmt.Errorf("get %t action error: %s", r, err)
		}
		if action != nil {
			actions = append(actions, action)
		}
	}

	err := e.RunAll(actions)
	if err != nil {
		return fmt.Errorf("running actions error: %s", err)
	}
	return nil
}
