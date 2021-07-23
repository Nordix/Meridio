package trench

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	"golang.org/x/net/context"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RoleBinding struct {
	currentStatus *rbacv1.RoleBinding
	desiredStatus *rbacv1.RoleBinding
}

func (r *RoleBinding) getModel() (*rbacv1.RoleBinding, error) {
	return common.GetRoleBindingModel("deployment/role-binding.yaml")
}

func (r *RoleBinding) insertParamters(role *rbacv1.RoleBinding, cr *meridiov1alpha1.Trench) *rbacv1.RoleBinding {
	role.ObjectMeta.Name = common.RoleBindingName(cr)
	role.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	role.Subjects[0].Name = common.ServiceAccountName(cr)
	role.RoleRef.Name = common.RoleName(cr)
	return role
}

func (r *RoleBinding) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      common.RoleBindingName(cr),
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

func (r *RoleBinding) getAction(e *common.Executor, cr *meridiov1alpha1.Trench) (common.Action, error) {
	var action common.Action
	err := r.getCurrentStatus(e.Ctx, cr, e.Client)
	if err != nil {
		return action, err
	}
	if r.currentStatus == nil {
		err = r.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.Log.Info("role binding", "add action", "create")
		action = common.NewCreateAction(r.desiredStatus, "create role binding")
	} else {
		r.getReconciledDesiredStatus(r.currentStatus, cr)
		if !equality.Semantic.DeepEqual(r.desiredStatus, r.currentStatus) {
			e.Log.Info("role binding", "add action", "update")
			action = common.NewUpdateAction(r.desiredStatus, "update role binding")
		}
	}
	return action, nil
}
