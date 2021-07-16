package reconciler

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nspName    = "nsp"
	nspEnvName = "NSP_PORT"
	imageNsp   = "nsp"
)

func getNSPDeploymentName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s-%s", nspName, cr.ObjectMeta.Name)
}

type NspDeployment struct {
	currentStatus *appsv1.Deployment
	desiredStatus *appsv1.Deployment
}

func (i *NspDeployment) getEnvVars(cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	return []corev1.EnvVar{
		{
			Name:  nspEnvName,
			Value: fmt.Sprint(nspTargetPort),
		},
	}
}

func (i *NspDeployment) insertParamters(dep *appsv1.Deployment, cr *meridiov1alpha1.Trench) *appsv1.Deployment {
	// if status nsp deployment parameters are specified in the cr, use those
	// else use the default parameters
	nspDeploymentName := getNSPDeploymentName(cr)
	dep.ObjectMeta.Name = nspDeploymentName
	dep.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	dep.ObjectMeta.Labels["app"] = nspDeploymentName
	dep.Spec.Selector.MatchLabels["app"] = nspDeploymentName
	dep.Spec.Template.ObjectMeta.Labels["app"] = nspDeploymentName
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, Organization, imageNsp, Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = PullPolicy
	dep.Spec.Template.Spec.Containers[0].LivenessProbe = GetLivenessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].ReadinessProbe = GetReadinessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(cr)
	return dep
}

func (i *NspDeployment) getModel() (*appsv1.Deployment, error) {
	return getDeploymentModel("deployment/nsp.yaml")
}

func (i *NspDeployment) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getNSPDeploymentName(cr),
	}
}

func (i *NspDeployment) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	NspDeployment, err := i.getModel()
	if err != nil {
		return err
	}
	i.desiredStatus = i.insertParamters(NspDeployment, cr)
	return nil
}

// getNspDeploymentReconciledDesiredStatus gets the desired status of nsp deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NspDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment, cr *meridiov1alpha1.Trench) {
	i.desiredStatus = i.insertParamters(cd, cr)
}

func (i *NspDeployment) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentStatus := &appsv1.Deployment{}
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

func (i *NspDeployment) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
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
		e.log.Info("nsp deployment", "add action", "create")
		action = newCreateAction(i.desiredStatus, "create nsp deployment")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.log.Info("nsp deployment", "add action", "update")
			action = newUpdateAction(i.desiredStatus, "update nsp deployment")
		}
	}
	return action, nil
}
