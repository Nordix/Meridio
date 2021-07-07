package reconciler

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	roleBindingName = "meridio-configuration-role-binding"
)

type RoleBinding struct {
	currentStatus *rbacv1.RoleBinding
	desiredStatus *rbacv1.RoleBinding
}

func (r *RoleBinding) getModel() (*rbacv1.RoleBinding, error) {
	return getRoleBindingModel("deployment/role-binding.yaml")
}

func (r *RoleBinding) insertParamters(role *rbacv1.RoleBinding, cr *meridiov1alpha1.Trench) *rbacv1.RoleBinding {
	role.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	return role
}

func (r *RoleBinding) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      roleBindingName,
	}
}

func (r *RoleBinding) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentState := &rbacv1.RoleBinding{}
	err := client.Get(ctx, r.getSelector(cr), currentState)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	r.currentStatus = currentState.DeepCopy()
	return nil
}

func (r *RoleBinding) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	model, err := r.getModel()
	if err != nil {
		return err
	}
	r.desiredStatus = r.insertParamters(model, cr)
	return nil
}

func (r *RoleBinding) getReconciledDesiredStatus(current *rbacv1.RoleBinding, cr *meridiov1alpha1.Trench) {
	r.desiredStatus = r.insertParamters(current, cr).DeepCopy()
}

func (r *RoleBinding) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
	var action Action
	err := r.getCurrentStatus(e.ctx, cr, e.client)
	if err != nil {
		return action, err
	}
	if r.currentStatus == nil {
		err = r.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.log.Info("role binding", "add action", "create")
		action = newCreateAction(r.desiredStatus, "create role binding")
	} else {
		r.getReconciledDesiredStatus(r.currentStatus, cr)
		if !equality.Semantic.DeepEqual(r.desiredStatus, r.currentStatus) {
			e.log.Info("role binding", "add action", "update")
			action = newUpdateAction(r.desiredStatus, "update role binding")
		}
	}
	return action, nil
}
