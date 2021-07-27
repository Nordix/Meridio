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
	currentStatus *rbacv1.Role
	desiredStatus *rbacv1.Role
}

func (r *Role) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      common.RoleName(cr),
	}
}

func (r *Role) getModel() (*rbacv1.Role, error) {
	return common.GetRoleModel("deployment/role.yaml")
}

func (r *Role) insertParamters(role *rbacv1.Role, cr *meridiov1alpha1.Trench) *rbacv1.Role {
	role.ObjectMeta.Name = common.RoleName(cr)
	role.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	return role
}

func (r *Role) getCurrentStatus(e *common.Executor, cr *meridiov1alpha1.Trench) error {
	currentStatus := &rbacv1.Role{}
	selector := r.getSelector(cr)
	err := e.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	r.currentStatus = currentStatus.DeepCopy()
	return nil
}

func (r *Role) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	model, err := r.getModel()
	if err != nil {
		return err
	}
	r.desiredStatus = r.insertParamters(model, cr)
	return nil
}

func (r *Role) getReconciledDesiredStatus(current *rbacv1.Role, cr *meridiov1alpha1.Trench) {
	r.desiredStatus = r.insertParamters(current, cr).DeepCopy()
}

func (r *Role) getAction(e *common.Executor, cr *meridiov1alpha1.Trench) (common.Action, error) {
	var action common.Action
	err := r.getCurrentStatus(e, cr)
	if err != nil {
		return action, err
	}
	if r.currentStatus == nil {
		err = r.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.LogInfo("add action: create role")
		action = common.NewCreateAction(r.desiredStatus, "create role")
	} else {
		r.getReconciledDesiredStatus(r.currentStatus, cr)
		if !equality.Semantic.DeepEqual(r.desiredStatus, r.currentStatus) {
			e.LogInfo("add action: update role")
			action = common.NewUpdateAction(r.desiredStatus, "update role")
		}
	}
	return action, nil
}
