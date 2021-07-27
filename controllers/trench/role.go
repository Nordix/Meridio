package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Role struct {
	trench *meridiov1alpha1.Trench
	model  *rbacv1.Role
}

func NewRole(t *meridiov1alpha1.Trench) (*Role, error) {
	l := &Role{
		trench: t.DeepCopy(),
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (r *Role) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: r.trench.ObjectMeta.Namespace,
		Name:      common.RoleName(r.trench),
	}
}

func (r *Role) getModel() error {
	model, err := common.GetRoleModel("deployment/role.yaml")
	if err != nil {
		return err
	}
	r.model = model
	return nil
}

func (r *Role) insertParamters(role *rbacv1.Role) *rbacv1.Role {
	ret := role.DeepCopy()
	ret.ObjectMeta.Name = common.RoleName(r.trench)
	ret.ObjectMeta.Namespace = r.trench.ObjectMeta.Namespace
	return ret
}

func (r *Role) getCurrentStatus(e *common.Executor) (*rbacv1.Role, error) {
	currentStatus := &rbacv1.Role{}
	selector := r.getSelector()
	err := e.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (r *Role) getDesiredStatus() *rbacv1.Role {
	return r.insertParamters(r.model)
}

func (r *Role) getReconciledDesiredStatus(current *rbacv1.Role) *rbacv1.Role {
	return r.insertParamters(current)
}

func (r *Role) getAction(e *common.Executor) (common.Action, error) {
	elem := common.RoleName(r.trench)
	var action common.Action
	cs, err := r.getCurrentStatus(e)
	if err != nil {
		return action, err
	}
	if cs == nil {
		ds := r.getDesiredStatus()
		e.LogInfo(fmt.Sprintf("add action: create %s", elem))
		action = common.NewCreateAction(ds, fmt.Sprintf("create %s", elem))
	} else {
		ds := r.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			e.LogInfo(fmt.Sprintf("add action: update %s", elem))
			action = common.NewUpdateAction(ds, fmt.Sprintf("update %s", elem))
		}
	}
	return action, nil
}
