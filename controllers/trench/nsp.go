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
	imageNsp = "nsp"
)

type NspDeployment struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.Deployment
	exec   *common.Executor
}

func NewNspDeployment(e *common.Executor, t *meridiov1alpha1.Trench) (*NspDeployment, error) {
	l := &NspDeployment{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *NspDeployment) getEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	ret := []corev1.EnvVar{}
	for _, env := range allEnv {
		switch env.Name {
		case "NSP_PORT":
			env.Value = fmt.Sprint(common.NspTargetPort)
		case "NSP_CONFIG_MAP_NAME":
			env.Value = common.ConfigMapName(i.trench)
		case "NSP_NAMESPACE":
		default:
			i.exec.LogError(fmt.Errorf("env %s not expected", env.Name), "get env var error")
		}
		ret = append(ret, env)
	}
	return ret
}

func (i *NspDeployment) insertParameters(init *appsv1.Deployment) *appsv1.Deployment {
	// if status nsp deployment parameters are specified in the cr, use those
	// else use the default parameters
	nspDeploymentName := common.NSPDeploymentName(i.trench)
	dep := init.DeepCopy()
	dep.ObjectMeta.Name = nspDeploymentName
	dep.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	dep.ObjectMeta.Labels["app"] = nspDeploymentName
	dep.Spec.Selector.MatchLabels["app"] = nspDeploymentName
	dep.Spec.Template.ObjectMeta.Labels["app"] = nspDeploymentName
	dep.Spec.Template.Spec.ServiceAccountName = common.ServiceAccountName(i.trench)

	dep.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()

	for k, container := range dep.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "nsp":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageNsp, common.Tag)
			}
			container.LivenessProbe = common.GetLivenessProbe(i.trench)
			container.ReadinessProbe = common.GetReadinessProbe(i.trench)
			container.Env = i.getEnvVars(container.Env)
		default:
			i.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		dep.Spec.Template.Spec.Containers[k] = container
	}

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
	return i.insertParameters(i.model)
}

// getNspDeploymentReconciledDesiredStatus gets the desired status of nsp deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment) *appsv1.Deployment {
	return i.insertParameters(cd)
}

func (i *NspDeployment) getCurrentStatus() (*appsv1.Deployment, error) {
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

func (i *NspDeployment) getAction() ([]common.Action, error) {
	var action []common.Action
	cs, err := i.getCurrentStatus()
	if err != nil {
		return nil, err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return nil, err
		}
		action = append(action, i.exec.NewCreateAction(ds))
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			action = append(action, i.exec.NewUpdateAction(ds))
		}
	}
	return action, nil
}
