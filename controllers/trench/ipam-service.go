package trench

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IpamService struct {
	currentStatus *corev1.Service
	desiredStatus *corev1.Service
}

func (i *IpamService) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      common.IPAMServiceName(cr),
	}
}

func (i *IpamService) insertParamters(svc *corev1.Service, cr *meridiov1alpha1.Trench) *corev1.Service {
	// if status ipam service parameters are specified in the cr, use those
	// else use the default parameters
	svc.ObjectMeta.Name = common.IPAMServiceName(cr)
	svc.Spec.Selector["app"] = common.IPAMDeploymentName(cr)
	svc.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	return svc
}

func (i *IpamService) getCurrentStatus(e *common.Executor, cr *meridiov1alpha1.Trench) error {
	currentStatus := &corev1.Service{}
	selector := i.getSelector(cr)
	err := e.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	i.currentStatus = currentStatus.DeepCopy()
	return nil
}

func (i *IpamService) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	ipams, err := i.getModel()
	if err != nil {
		return err
	}
	i.desiredStatus = i.insertParamters(ipams, cr)
	return nil
}

// getReconciledDesiredStatus gets the desired status of ipam service after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *IpamService) getReconciledDesiredStatus(ipams *corev1.Service, cr *meridiov1alpha1.Trench) {
	i.desiredStatus = i.insertParamters(ipams, cr)
}

func (i *IpamService) getModel() (*corev1.Service, error) {
	return common.GetServiceModel("deployment/ipam-service.yaml")
}

func (i *IpamService) getAction(e *common.Executor, cr *meridiov1alpha1.Trench) (common.Action, error) {
	var action common.Action
	err := i.getCurrentStatus(e, cr)
	if err != nil {
		return nil, err
	}
	if i.currentStatus == nil {
		err = i.getDesiredStatus(cr)
		if err != nil {
			return nil, err
		}
		e.LogInfo("add action: create ipam service")
		action = common.NewCreateAction(i.desiredStatus, "create ipam service")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.LogInfo("add action: update ipam service")
			action = common.NewUpdateAction(i.desiredStatus, "update ipam service")
		}
	}
	return action, nil
}
