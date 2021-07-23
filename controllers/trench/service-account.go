package trench

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const serviceAccountName = "meridio"

type ServiceAccount struct {
	currentStatus *corev1.ServiceAccount
	desiredStatus *corev1.ServiceAccount
}

func getServiceAccountName(cr *meridiov1alpha1.Trench) string {
	return common.GetFullName(cr, serviceAccountName)
}

func (sa *ServiceAccount) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getServiceAccountName(cr),
	}
}

func (sa *ServiceAccount) insertParamters(role *corev1.ServiceAccount, cr *meridiov1alpha1.Trench) *corev1.ServiceAccount {
	role.ObjectMeta.Name = getServiceAccountName(cr)
	role.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	return role
}

func (sa *ServiceAccount) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentState := &corev1.ServiceAccount{}
	err := client.Get(ctx, sa.getSelector(cr), currentState)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	sa.currentStatus = sa.insertParamters(currentState, cr)
	return nil
}

func (sa *ServiceAccount) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	sa.desiredStatus = &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getServiceAccountName(cr),
			Namespace: cr.ObjectMeta.Namespace,
		},
	}
	return nil
}

func (sa *ServiceAccount) getReconciledDesiredStatus(current *corev1.ServiceAccount, cr *meridiov1alpha1.Trench) {
	sa.desiredStatus = sa.insertParamters(current, cr).DeepCopy()
}

func (sa *ServiceAccount) getAction(e *common.Executor, cr *meridiov1alpha1.Trench) (common.Action, error) {
	var action common.Action
	err := sa.getCurrentStatus(e.Ctx, cr, e.Client)
	if err != nil {
		return action, err
	}
	if sa.currentStatus == nil {
		err = sa.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.Log.Info("service account", "add action", "create")
		action = common.NewCreateAction(sa.desiredStatus, "create service account")
	} else {
		sa.getReconciledDesiredStatus(sa.currentStatus, cr)
		if !equality.Semantic.DeepEqual(sa.desiredStatus, sa.currentStatus) {
			e.Log.Info("service account", "add action", "update")
			action = common.NewUpdateAction(sa.desiredStatus, "update service account")
		}
	}
	return action, nil
}
