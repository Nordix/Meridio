package trench

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RoleBinding struct {
	trench *meridiov1alpha1.Trench
	model  *rbacv1.RoleBinding
	exec   *common.Executor
}

func NewRoleBinding(e *common.Executor, t *meridiov1alpha1.Trench) (*RoleBinding, error) {
	l := &RoleBinding{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (r *RoleBinding) getModel() error {
	model, err := common.GetRoleBindingModel("deployment/role-binding.yaml")
	if err != nil {
		return err
	}
	r.model = model
	return nil
}

func (r *RoleBinding) insertParameters(init *rbacv1.RoleBinding) *rbacv1.RoleBinding {
	ret := init.DeepCopy()
	ret.ObjectMeta.Name = common.RoleBindingName(r.trench)
	ret.ObjectMeta.Namespace = r.trench.ObjectMeta.Namespace
	if len(ret.Subjects) != 0 {
		ret.Subjects[0].Name = common.ServiceAccountName(r.trench)
	} else {
		ret.Subjects = []rbacv1.Subject{
			{
				Kind: "",
				Name: common.ServiceAccountName(r.trench),
			},
		}
	}
	ret.RoleRef.Name = common.RoleName(r.trench)
	return ret
}

func (r *RoleBinding) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: r.trench.ObjectMeta.Namespace,
		Name:      common.RoleBindingName(r.trench),
	}
}

func (r *RoleBinding) getCurrentStatus() (*rbacv1.RoleBinding, error) {
	currentStatus := &rbacv1.RoleBinding{}
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

func (r *RoleBinding) getDesiredStatus() *rbacv1.RoleBinding {
	return r.insertParameters(r.model)
}

func (r *RoleBinding) getReconciledDesiredStatus(cd *rbacv1.RoleBinding) *rbacv1.RoleBinding {
	template := cd.DeepCopy()
	template.RoleRef = r.model.RoleRef
	template.Subjects = r.model.Subjects
	return r.insertParameters(template)
}

func (r *RoleBinding) getAction() error {
	cs, err := r.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := r.getDesiredStatus()
		r.exec.AddCreateAction(ds)
	} else {
		ds := r.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			r.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
