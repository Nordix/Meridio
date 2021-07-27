package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nspEnvName = "NSP_PORT"
	imageNsp   = "nsp"
)

type NspDeployment struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.Deployment
}

func NewNspDeployment(t *meridiov1alpha1.Trench) (*NspDeployment, error) {
	l := &NspDeployment{
		trench: t.DeepCopy(),
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *NspDeployment) getEnvVars() []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	return []corev1.EnvVar{
		{
			Name:  nspEnvName,
			Value: fmt.Sprint(common.NspTargetPort),
		},
	}
}

func (i *NspDeployment) insertParamters(init *appsv1.Deployment) *appsv1.Deployment {
	// if status nsp deployment parameters are specified in the cr, use those
	// else use the default parameters
	nspDeploymentName := common.NSPDeploymentName(i.trench)
	dep := init.DeepCopy()
	dep.ObjectMeta.Name = nspDeploymentName
	dep.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	dep.ObjectMeta.Labels["app"] = nspDeploymentName
	dep.Spec.Selector.MatchLabels["app"] = nspDeploymentName
	dep.Spec.Template.ObjectMeta.Labels["app"] = nspDeploymentName
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageNsp, common.Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = common.PullPolicy
	dep.Spec.Template.Spec.Containers[0].LivenessProbe = common.GetLivenessProbe(i.trench)
	dep.Spec.Template.Spec.Containers[0].ReadinessProbe = common.GetReadinessProbe(i.trench)
	dep.Spec.Template.Spec.Containers[0].Env = i.getEnvVars()
	return dep
}

func (i *NspDeployment) getModel() error {
	model, err := common.GetDeploymentModel("deployment/nsp.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *NspDeployment) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.NSPDeploymentName(i.trench),
	}
}

func (i *NspDeployment) getDesiredStatus() *appsv1.Deployment {
	return i.insertParamters(i.model)
}

// getNspDeploymentReconciledDesiredStatus gets the desired status of nsp deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment) *appsv1.Deployment {
	return i.insertParamters(cd)
}

func (i *NspDeployment) getCurrentStatus(e *common.Executor) (*appsv1.Deployment, error) {
	currentStatus := &appsv1.Deployment{}
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

func (i *NspDeployment) getAction(e *common.Executor) (common.Action, error) {
	elem := common.NSPDeploymentName(i.trench)
	var action common.Action
	cs, err := i.getCurrentStatus(e)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return nil, err
		}
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
