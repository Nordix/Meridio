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
	ipamStatefulSet *IpamStatefulSet
	ipamService     *IpamService
	serviceAccount  *ServiceAccount
	role            *Role
	roleBinding     *RoleBinding
	nspStatefulSet  *NspStatefulSet
	nspService      *NspService
	configmap       *ConfigMap
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
	nspd, err := NewNspStatefulSet(e, trench)
	if err != nil {
		return nil, err
	}
	nsps, err := NewNspService(e, trench)
	if err != nil {
		return nil, err
	}
	cfg := NewConfigMap(e, trench)
	return &Meridio{
		ipamStatefulSet: ipam,
		ipamService:     ipamsvc,
		serviceAccount:  sa,
		role:            role,
		roleBinding:     rb,
		nspStatefulSet:  nspd,
		nspService:      nsps,
		configmap:       cfg,
	}, nil
}

func (m Meridio) ReconcileAll() error {
	resources := []Resources{
		m.serviceAccount,
		m.role,
		m.roleBinding,
		m.nspStatefulSet,
		m.nspService,
		m.ipamStatefulSet,
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
