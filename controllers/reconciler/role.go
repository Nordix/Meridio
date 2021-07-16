package reconciler

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	roleName = "meridio-configuration-role"
)

func getRoleName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s-%s", roleName, cr.ObjectMeta.Name)
}

type Role struct {
	currentStatus *rbacv1.Role
	desiredStatus *rbacv1.Role
}

func (r *Role) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getRoleName(cr),
	}
}

func (r *Role) getModel() (*rbacv1.Role, error) {
	return getRoleModel("deployment/role.yaml")
}

func (r *Role) insertParamters(role *rbacv1.Role, cr *meridiov1alpha1.Trench) *rbacv1.Role {
	role.ObjectMeta.Name = getRoleName(cr)
	role.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	return role
}

func (r *Role) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentState := &rbacv1.Role{}
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

func (r *Role) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
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
		e.log.Info("role", "add action", "create")
		action = newCreateAction(r.desiredStatus, "create role")
	} else {
		r.getReconciledDesiredStatus(r.currentStatus, cr)
		if !equality.Semantic.DeepEqual(r.desiredStatus, r.currentStatus) {
			e.log.Info("role", "add action", "update")
			action = newUpdateAction(r.desiredStatus, "update role")
		}
	}
	return action, nil
}
