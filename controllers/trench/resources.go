package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
)

type Resources interface {
	getAction() error
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
	configmap      *ConfigMap
}

func NewMeridio(e *common.Executor, trench *meridiov1alpha1.Trench) (*Meridio, error) {
	ipamsvc, err := NewIPAMSvc(e, trench)
	if err != nil {
		return nil, err
	}
	ipam, err := NewIPAM(e, trench)
	if err != nil {
		return nil, err
	}
	sa, err := NewServiceAccount(e, trench)
	if err != nil {
		return nil, err
	}
	role, err := NewRole(e, trench)
	if err != nil {
		return nil, err
	}
	rb, err := NewRoleBinding(e, trench)
	if err != nil {
		return nil, err
	}
	nspd, err := NewNspDeployment(e, trench)
	if err != nil {
		return nil, err
	}
	nsps, err := NewNspService(e, trench)
	if err != nil {
		return nil, err
	}
	p, err := NewProxy(e, trench)
	if err != nil {
		return nil, err
	}
	cfg := NewConfigMap(e, trench)
	return &Meridio{
		ipamDeployment: ipam,
		ipamService:    ipamsvc,
		serviceAccount: sa,
		role:           role,
		roleBinding:    rb,
		nspDeployment:  nspd,
		nspService:     nsps,
		proxy:          p,
		configmap:      cfg,
	}, nil
}

func (m Meridio) ReconcileAll() error {
	resources := []Resources{
		m.serviceAccount,
		m.role,
		m.roleBinding,
		m.nspDeployment,
		m.nspService,
		m.proxy,
		m.ipamDeployment,
		m.ipamService,
		m.configmap,
	}

	for _, r := range resources {
		err := r.getAction()
		if err != nil {
			return fmt.Errorf("get %t action error: %s", r, err)
		}
	}
	return nil
}
