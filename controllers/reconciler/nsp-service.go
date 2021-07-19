package reconciler

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nspPort       = 7778
	nspTargetPort = 7778
	nspSvcName    = "nsp-service"
)

func getNSPServiceName(cr *meridiov1alpha1.Trench) string {
	return getFullName(cr, nspSvcName)
}

type NspService struct {
	currentStatus *corev1.Service
	desiredStatus *corev1.Service
}

func (i *NspService) getPorts(cr *meridiov1alpha1.Trench) []corev1.ServicePort {
	// if nsp service ports are set in the cr, use the values
	// else return default service ports
	return []corev1.ServicePort{
		{
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(nspTargetPort),
			Port:       nspPort,
		},
	}
}

func (i *NspService) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getNSPServiceName(cr),
	}
}

func (i *NspService) insertParamters(svc *corev1.Service, cr *meridiov1alpha1.Trench) *corev1.Service {
	// if status nsp service parameters are specified in the cr, use those
	// else use the default parameters
	svc.ObjectMeta.Name = getNSPServiceName(cr)
	svc.Spec.Selector["app"] = getNSPDeploymentName(cr)
	svc.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	svc.Spec.Ports = i.getPorts(cr)
	return svc
}

func (i *NspService) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentStatus := &corev1.Service{}
	selector := i.getSelector(cr)
	err := client.Get(ctx, selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	i.currentStatus = currentStatus.DeepCopy()
	return nil
}

func (i *NspService) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	svc, err := i.getModel()
	if err != nil {
		return err
	}
	i.desiredStatus = i.insertParamters(svc, cr)
	return nil
}

// getReconciledDesiredStatus gets the desired status of nsp service after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspService) getReconciledDesiredStatus(svc *corev1.Service, cr *meridiov1alpha1.Trench) {
	svc = i.insertParamters(svc, cr)
	i.desiredStatus = svc
}

func (i *NspService) getModel() (*corev1.Service, error) {
	return getServiceModel("deployment/nsp-service.yaml")
}

func (i *NspService) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
	var action Action
	err := i.getCurrentStatus(e.ctx, cr, e.client)
	if err != nil {
		return action, err
	}
	if i.currentStatus == nil {
		err := i.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.log.Info("nsp service", "add action", "create")
		action = newCreateAction(i.desiredStatus, "create nsp service")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.log.Info("nsp service", "add action", "update")
			action = newUpdateAction(i.desiredStatus, "update nsp service")
		}
	}
	return action, nil
}
