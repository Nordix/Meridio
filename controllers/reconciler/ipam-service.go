package reconciler

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ipamPort       = 7777
	ipamTargetPort = 7777
	ipamSvcName    = "ipam-service"
)

func getIPAMServiceName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s-%s", ipamSvcName, cr.ObjectMeta.Name)
}

type IpamService struct {
	currentStatus *corev1.Service
	desiredStatus *corev1.Service
}

func (i *IpamService) getPorts(cr *meridiov1alpha1.Trench) []corev1.ServicePort {
	// if ipam service ports are set in the cr, use the values
	// else return default service ports
	return []corev1.ServicePort{
		{
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(ipamTargetPort),
			Port:       ipamPort,
		},
	}
}

func (i *IpamService) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getIPAMServiceName(cr),
	}
}

func (i *IpamService) insertParamters(svc *corev1.Service, cr *meridiov1alpha1.Trench) *corev1.Service {
	// if status ipam service parameters are specified in the cr, use those
	// else use the default parameters
	svc.ObjectMeta.Name = getIPAMServiceName(cr)
	svc.Spec.Selector["app"] = getIPAMDeploymentName(cr)
	svc.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	svc.Spec.Ports = i.getPorts(cr)
	return svc
}

func (i *IpamService) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
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
	return getServiceModel("deployment/ipam-service.yaml")
}

func (i *IpamService) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
	var action Action
	err := i.getCurrentStatus(e.ctx, cr, e.client)
	if err != nil {
		return nil, err
	}
	if i.currentStatus == nil {
		err = i.getDesiredStatus(cr)
		if err != nil {
			return nil, err
		}
		e.log.Info("ipam service", "add action", "create")
		action = newCreateAction(i.desiredStatus, "create ipam service")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.log.Info("ipam service", "add action", "update")
			action = newUpdateAction(i.desiredStatus, "update ipam service")
		}
	}
	return action, nil
}
