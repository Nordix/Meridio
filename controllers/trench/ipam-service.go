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
	trench *meridiov1alpha1.Trench
	model  *corev1.Service
	exec   *common.Executor
}

func NewIPAMSvc(e *common.Executor, t *meridiov1alpha1.Trench) (*IpamService, error) {
	l := &IpamService{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *IpamService) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.IPAMServiceName(i.trench),
	}
}

func (i *IpamService) insertParameters(svc *corev1.Service) *corev1.Service {
	// if status ipam service parameters are specified in the cr, use those
	// else use the default parameters
	ret := svc.DeepCopy()
	ret.ObjectMeta.Name = common.IPAMServiceName(i.trench)
	ret.Spec.Selector["app"] = common.IPAMStatefulSetName(i.trench)
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	return ret
}

func (i *IpamService) getCurrentStatus() (*corev1.Service, error) {
	currentStatus := &corev1.Service{}
	selector := i.getSelector()
	err := i.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (i *IpamService) getDesiredStatus() *corev1.Service {
	return i.insertParameters(i.model)
}

// getReconciledDesiredStatus gets the desired status of ipam service after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *IpamService) getReconciledDesiredStatus(svc *corev1.Service) *corev1.Service {
	template := svc.DeepCopy()
	template.Spec.Type = i.model.Spec.Type
	return i.insertParameters(svc)
}

func (i *IpamService) getModel() error {
	model, err := common.GetServiceModel("deployment/ipam-service.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *IpamService) getAction() error {
	cs, err := i.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
