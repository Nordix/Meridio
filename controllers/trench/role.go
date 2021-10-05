package trench

import (
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
	exec   *common.Executor
}

func NewRole(e *common.Executor, t *meridiov1alpha1.Trench) (*Role, error) {
	l := &Role{
		trench: t.DeepCopy(),
		exec:   e,
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

func (r *Role) insertParameters(role *rbacv1.Role) *rbacv1.Role {
	ret := role.DeepCopy()
	ret.ObjectMeta.Name = common.RoleName(r.trench)
	ret.ObjectMeta.Namespace = r.trench.ObjectMeta.Namespace
	return ret
}

func (r *Role) getCurrentStatus() (*rbacv1.Role, error) {
	currentStatus := &rbacv1.Role{}
	selector := r.getSelector()
	err := r.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (r *Role) getDesiredStatus() *rbacv1.Role {
	return r.insertParameters(r.model)
}

func (r *Role) getReconciledDesiredStatus(current *rbacv1.Role) *rbacv1.Role {
	return r.insertParameters(current)
}

func (r *Role) getAction() ([]common.Action, error) {
	var action []common.Action
	cs, err := r.getCurrentStatus()
	if err != nil {
		return action, err
	}
	if cs == nil {
		ds := r.getDesiredStatus()
		action = append(action, r.exec.NewCreateAction(ds))
	} else {
		ds := r.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			action = append(action, r.exec.NewUpdateAction(ds))
		}
	}
	return action, nil
}
