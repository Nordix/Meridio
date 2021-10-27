package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imageIpam   = "ipam"
	ipamEnvName = "IPAM_PORT"
)

type IpamDeployment struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.Deployment
	exec   *common.Executor
}

func NewIPAM(e *common.Executor, t *meridiov1alpha1.Trench) (*IpamDeployment, error) {
	l := &IpamDeployment{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *IpamDeployment) insertParameters(dep *appsv1.Deployment) *appsv1.Deployment {
	// if status ipam deployment parameters are specified in the cr, use those
	// else use the default parameters
	ret := dep.DeepCopy()
	ipamDeploymentName := common.IPAMDeploymentName(i.trench)
	ret.ObjectMeta.Name = ipamDeploymentName
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = ipamDeploymentName
	ret.Spec.Selector.MatchLabels["app"] = ipamDeploymentName
	ret.Spec.Template.ObjectMeta.Labels["app"] = ipamDeploymentName
	if ret.Spec.Template.Spec.Containers[0].Image == "" {
		ret.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageIpam, common.Tag)
	}
	ret.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()
	ret.Spec.Template.Spec.Containers[0].LivenessProbe = common.GetLivenessProbe(i.trench)
	ret.Spec.Template.Spec.Containers[0].ReadinessProbe = common.GetReadinessProbe(i.trench)
	return ret
}

func (i *IpamDeployment) getModel() error {
	model, err := common.GetDeploymentModel("deployment/ipam.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *IpamDeployment) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.IPAMDeploymentName(i.trench),
	}
}

func (i *IpamDeployment) getDesiredStatus() *appsv1.Deployment {
	return i.insertParameters(i.model)
}

// getIpamDeploymentReconciledDesiredStatus gets the desired status of ipam deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *IpamDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment) *appsv1.Deployment {
	return i.insertParameters(cd)
}

func (i *IpamDeployment) getCurrentStatus() (*appsv1.Deployment, error) {
	currentStatus := &appsv1.Deployment{}
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

func (i *IpamDeployment) getAction() error {
	cs, err := i.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return err
		}
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
