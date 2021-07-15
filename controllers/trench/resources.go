package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Resources interface {
	getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error
	getDesiredStatus(cr *meridiov1alpha1.Trench) error
	getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error)
}

type Meridio struct {
	ipamDeployment *IpamDeployment
	ipamService    *IpamService
	configmap      *ConfigMap
	loadBalancer   *LoadBalancer
	serviceAccount *ServiceAccount
	role           *Role
	roleBinding    *RoleBinding
	nspDeployment  *NspDeployment
	nspService     *NspService
	proxy          *Proxy
	nseDeployment  *NseDeployment
}

func NewMeridio() *Meridio {
	return &Meridio{
		ipamDeployment: &IpamDeployment{},
		ipamService:    &IpamService{},
		configmap:      &ConfigMap{},
		loadBalancer:   &LoadBalancer{},
		serviceAccount: &ServiceAccount{},
		role:           &Role{},
		roleBinding:    &RoleBinding{},
		nspDeployment:  &NspDeployment{},
		nspService:     &NspService{},
		proxy:          &Proxy{},
		nseDeployment:  &NseDeployment{},
	}
}

func (m Meridio) ReconcileAll(e *Executor, cr *meridiov1alpha1.Trench) error {
	var actions []Action
	resources := []Resources{
		m.ipamDeployment,
		m.ipamService,
		m.configmap,
		m.serviceAccount,
		m.role,
		m.roleBinding,
		m.loadBalancer,
		m.nspDeployment,
		m.nspService,
		m.proxy,
		m.nseDeployment,
	}

	for _, r := range resources {
		action, err := r.getAction(e, cr)
		if err != nil {
			return fmt.Errorf("get %t action error: %s", r, err)
		}
		if action != nil {
			actions = append(actions, action)
		}
	}

	err := e.runAll(actions)
	if err != nil {
		return fmt.Errorf("running actions error: %s", err)
	}
	return nil
}
