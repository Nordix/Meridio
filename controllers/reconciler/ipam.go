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
	nameIpam    = "ipam"
	imageIpam   = "ipam"
	ipamEnvName = "IPAM_PORT"
)

func getIPAMDeploymentName(cr *meridiov1alpha1.Trench) string {
	return getFullName(cr, nameIpam)
}

type IpamDeployment struct {
	currentStatus *appsv1.Deployment
	desiredStatus *appsv1.Deployment
}

func (i *IpamDeployment) getEnvVars(cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	return []corev1.EnvVar{
		{
			Name:  ipamEnvName,
			Value: fmt.Sprint(ipamTargetPort),
		},
	}
}

func (i *IpamDeployment) insertParamters(dep *appsv1.Deployment, cr *meridiov1alpha1.Trench) *appsv1.Deployment {
	// if status ipam deployment parameters are specified in the cr, use those
	// else use the default parameters
	ipamDeploymentName := getIPAMDeploymentName(cr)
	dep.ObjectMeta.Name = ipamDeploymentName
	dep.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	dep.ObjectMeta.Labels["app"] = ipamDeploymentName
	dep.Spec.Selector.MatchLabels["app"] = ipamDeploymentName
	dep.Spec.Template.ObjectMeta.Labels["app"] = ipamDeploymentName
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, Organization, imageIpam, Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = PullPolicy
	dep.Spec.Template.Spec.Containers[0].LivenessProbe = GetLivenessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].ReadinessProbe = GetReadinessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(cr)
	return dep
}

func (i *IpamDeployment) getModel() (*appsv1.Deployment, error) {
	return getDeploymentModel("deployment/ipam.yaml")
}

func (i *IpamDeployment) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getIPAMDeploymentName(cr),
	}
}

func (i *IpamDeployment) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	ipamDeployment, err := i.getModel()
	if err != nil {
		return err
	}
	i.desiredStatus = i.insertParamters(ipamDeployment, cr)
	return nil
}

// getIpamDeploymentReconciledDesiredStatus gets the desired status of ipam deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *IpamDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment, cr *meridiov1alpha1.Trench) {
	i.desiredStatus = i.insertParamters(cd, cr)
}

func (i *IpamDeployment) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
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

func (i *IpamDeployment) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
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
		e.log.Info("ipam deployment", "add action", "create")
		action = newCreateAction(i.desiredStatus, "create ipam deployment")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {

			e.log.Info("ipam deployment", "add action", "update")
			action = newUpdateAction(i.desiredStatus, "update ipam deployment")
		}
	}
	return action, nil
}
