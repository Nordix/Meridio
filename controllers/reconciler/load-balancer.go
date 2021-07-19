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
	lbName                  = "load-balancer"
	lbImage                 = "load-balancer"
	lbEnvConfig             = "NSM_CONFIG_MAP_NAME"
	lbEnvServiceName        = "NSM_SERVICE_NAME"
	lbEnvNsp                = "NSM_NSP_SERVICE"
	lbnscEnvNetworkServices = "NSM_NETWORK_SERVICES"
	lbfeEnvConfig           = "NFE_CONFIG_MAP_NAME"
)

func getLoadBalancerDeploymentName(cr *meridiov1alpha1.Trench) string {
	return getFullName(cr, lbName)
}

type LoadBalancer struct {
	currentStatus *appsv1.Deployment
	desiredStatus *appsv1.Deployment
}

func (l *LoadBalancer) setEnvVars(dep *appsv1.Deployment, cr *meridiov1alpha1.Trench) {
	// load-balancer container
	env := []corev1.EnvVar{
		{
			Name:  lbEnvConfig,
			Value: getConfigMapName(cr),
		},
		{
			Name:  lbEnvServiceName,
			Value: getLoadBalancerNsName(cr),
		},
		{
			Name:  lbEnvNsp,
			Value: getNSPService(cr),
		},
	}
	for _, e := range dep.Spec.Template.Spec.Containers[0].Env {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_NAMESPACE" ||
			e.Name == "NSM_CONNECT_TO" {
			env = append(env, e)
		}
	}
	dep.Spec.Template.Spec.Containers[0].Env = env
	// nsc container
	env = []corev1.EnvVar{
		{
			Name:  lbnscEnvNetworkServices,
			Value: fmt.Sprintf("vlan://%s.%s.%s/ext-vlan?forwarder=forwarder-vlan", vlanNetworkService, cr.ObjectMeta.Name, cr.ObjectMeta.Namespace),
		},
	}
	for _, e := range dep.Spec.Template.Spec.Containers[1].Env {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_DIAL_TIMEOUT" ||
			e.Name == "NSM_REQUEST_TIMEOUT" {
			env = append(env, e)
		}
	}
	dep.Spec.Template.Spec.Containers[1].Env = env
	// fe container
	env = []corev1.EnvVar{
		{
			Name:  lbfeEnvConfig,
			Value: getConfigMapName(cr),
		},
	}
	for _, e := range dep.Spec.Template.Spec.Containers[2].Env {
		// append all hard coded envVars
		if e.Name == "NFE_NAMESPACE" ||
			e.Name == "NFE_GATEWAYS" ||
			e.Name == "NFE_LOG_BIRD" ||
			e.Name == "NFE_ECMP" {
			env = append(env, e)
		}
	}
	dep.Spec.Template.Spec.Containers[2].Env = env
}

func (l *LoadBalancer) getModel() (*appsv1.Deployment, error) {
	return getDeploymentModel("deployment/load-balancer.yaml")
}

func (l *LoadBalancer) insertParamters(dep *appsv1.Deployment, cr *meridiov1alpha1.Trench) *appsv1.Deployment {
	// if status load-balancer deployment parameters are specified in the cr, use those
	// else use the default parameters
	loadBalancerDeploymentName := getLoadBalancerDeploymentName(cr)
	dep.ObjectMeta.Name = loadBalancerDeploymentName
	dep.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	dep.ObjectMeta.Labels["app"] = loadBalancerDeploymentName
	dep.Spec.Selector.MatchLabels["app"] = loadBalancerDeploymentName
	dep.Spec.Template.ObjectMeta.Labels["app"] = loadBalancerDeploymentName
	dep.Spec.Template.Spec.ServiceAccountName = getServiceAccountName(cr)
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, Organization, lbImage, Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = PullPolicy
	dep.Spec.Template.Spec.Containers[0].LivenessProbe = GetLivenessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].ReadinessProbe = GetReadinessProbe(cr)
	l.setEnvVars(dep, cr)
	return dep
}

func (l *LoadBalancer) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getLoadBalancerDeploymentName(cr),
	}
}

func (l *LoadBalancer) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentState := &appsv1.Deployment{}
	err := client.Get(ctx, l.getSelector(cr), currentState)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	l.currentStatus = currentState.DeepCopy()
	return nil
}

func (l *LoadBalancer) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	dep, err := l.getModel()
	if err != nil {
		return err
	}
	l.desiredStatus = l.insertParamters(dep, cr)
	return nil
}

// getReconciledDesiredStatus gets the desired status of load-balancer deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *LoadBalancer) getReconciledDesiredStatus(lb *appsv1.Deployment, cr *meridiov1alpha1.Trench) {
	lb = i.insertParamters(lb, cr)
	i.desiredStatus = lb
}

func (l *LoadBalancer) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
	var action Action
	err := l.getCurrentStatus(e.ctx, cr, e.client)
	if err != nil {
		return nil, err
	}
	if l.currentStatus == nil {
		err = l.getDesiredStatus(cr)
		if err != nil {
			return nil, err
		}
		e.log.Info("load-balancer", "add action", "create")
		action = newCreateAction(l.desiredStatus, "create load-balncer deployment")
	} else {
		l.getReconciledDesiredStatus(l.currentStatus, cr)
		if !equality.Semantic.DeepEqual(l.desiredStatus, l.currentStatus) {
			e.log.Info("load-balancer", "add action", "update")
			action = newUpdateAction(l.desiredStatus, "update load-balncer deployment")
		}
	}
	return action, nil
}
