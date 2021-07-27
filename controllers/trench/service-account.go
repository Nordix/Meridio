package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceAccount struct {
	trench *meridiov1alpha1.Trench
	model  *corev1.ServiceAccount
}

func NewServiceAccount(t *meridiov1alpha1.Trench) (*ServiceAccount, error) {
	l := &ServiceAccount{
		trench: t.DeepCopy(),
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (sa *ServiceAccount) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: sa.trench.ObjectMeta.Namespace,
		Name:      common.ServiceAccountName(sa.trench),
	}
}

func (i *ServiceAccount) getModel() error {
	model, err := common.GetServiceAccountModel("deployment/service-account.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (sa *ServiceAccount) insertParamters(init *corev1.ServiceAccount) *corev1.ServiceAccount {
	ret := init.DeepCopy()
	ret.ObjectMeta.Name = common.ServiceAccountName(sa.trench)
	ret.ObjectMeta.Namespace = sa.trench.ObjectMeta.Namespace
	return ret
}

func (sa *ServiceAccount) getCurrentStatus(e *common.Executor) (*corev1.ServiceAccount, error) {
	currentState := &corev1.ServiceAccount{}
	selector := sa.getSelector()
	err := e.GetObject(selector, currentState)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return sa.insertParamters(currentState), nil
}

func (sa *ServiceAccount) getDesiredStatus() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ServiceAccountName(sa.trench),
			Namespace: sa.trench.ObjectMeta.Namespace,
		},
	}
}

func (sa *ServiceAccount) getReconciledDesiredStatus(current *corev1.ServiceAccount) *corev1.ServiceAccount {
	return sa.insertParamters(current)
}

func (sa *ServiceAccount) getAction(e *common.Executor) (common.Action, error) {
	elem := common.ServiceAccountName(sa.trench)
	var action common.Action
	cs, err := sa.getCurrentStatus(e)
	if err != nil {
		return action, err
	}
	if cs == nil {
		ds := sa.getDesiredStatus()
		e.LogInfo(fmt.Sprintf("add action: create %s", elem))
		action = common.NewCreateAction(ds, fmt.Sprintf("create %s", elem))
	} else {
		ds := sa.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			e.LogInfo(fmt.Sprintf("add action: update %s", elem))
			action = common.NewUpdateAction(ds, fmt.Sprintf("update %s", elem))
		}
	}
	return action, nil
}
