package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NspService struct {
	trench *meridiov1alpha1.Trench
	model  *corev1.Service
}

func NewNspService(t *meridiov1alpha1.Trench) (*NspService, error) {
	l := &NspService{
		trench: t.DeepCopy(),
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *NspService) getPorts() []corev1.ServicePort {
	// if nsp service ports are set in the cr, use the values
	// else return default service ports
	return []corev1.ServicePort{
		{
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(common.NspTargetPort),
			Port:       common.NspPort,
		},
	}
}

func (i *NspService) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.NSPServiceName(i.trench),
	}
}

func (i *NspService) insertParamters(svc *corev1.Service) *corev1.Service {
	// if status nsp service parameters are specified in the cr, use those
	// else use the default parameters
	ret := svc.DeepCopy()
	ret.ObjectMeta.Name = common.NSPServiceName(i.trench)
	ret.Spec.Selector["app"] = common.NSPDeploymentName(i.trench)
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ret.Spec.Ports = i.getPorts()
	return ret
}

func (i *NspService) getCurrentStatus(e *common.Executor) (*corev1.Service, error) {
	currentStatus := &corev1.Service{}
	selector := i.getSelector()
	err := e.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (i *NspService) getDesiredStatus() *corev1.Service {
	return i.insertParamters(i.model)
}

// getReconciledDesiredStatus gets the desired status of nsp service after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspService) getReconciledDesiredStatus(svc *corev1.Service) *corev1.Service {
	return i.insertParamters(svc)
}

func (i *NspService) getModel() error {
	model, err := common.GetServiceModel("deployment/nsp-service.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *NspService) getAction(e *common.Executor) (common.Action, error) {
	elem := common.NSPServiceName(i.trench)
	var action common.Action
	cs, err := i.getCurrentStatus(e)
	if err != nil {
		return action, err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		e.LogInfo(fmt.Sprintf("add action: create %s", elem))
		action = common.NewCreateAction(ds, fmt.Sprintf("create %s", elem))
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			e.LogInfo(fmt.Sprintf("add action: update %s", elem))
			action = common.NewUpdateAction(ds, fmt.Sprintf("update %s", elem))
		}
	}
	return action, nil
}
