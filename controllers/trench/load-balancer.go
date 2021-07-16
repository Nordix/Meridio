package trench

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lbImage      = "load-balancer"
	lbName       = "load-balancer"
	lbEnvConfig  = "NSM_CONFIG_MAP_NAME"
	lbEnvService = "NSM_SERVICE_NAME"
	lbEnvNsp     = "NSM_NSP_SERVICE"
	nscEnvNwSvc  = "NSM_NETWORK_SERVICES"
	feEnvConfig  = "NFE_CONFIG_MAP_NAME"
)

func getLoadBalancerDeploymentName(cr *meridiov1alpha1.Trench) string {
	return common.GetFullName(cr, lbName)
}

type LoadBalancer struct {
	currentStatus *appsv1.Deployment
	desiredStatus *appsv1.Deployment
}

func (l *LoadBalancer) getModel() (*appsv1.Deployment, error) {
	return common.GetDeploymentModel("deployment/load-balancer.yaml")
}

func (l *LoadBalancer) getLbEnvVars(con *corev1.Container, cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  lbEnvConfig,
			Value: common.GetConfigMapName(cr),
		},
		{
			Name:  lbEnvService,
			Value: common.GetAppNsName(lbName, cr),
		},
		{
			Name:  lbEnvNsp,
			Value: getNSPServiceWithPort(cr),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_NAMESPACE" ||
			e.Name == "NSM_CONNECT_TO" {
			env = append(env, e)
		}
	}
	return env
}

func (l *LoadBalancer) getNscEnvVars(con *corev1.Container, cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  nscEnvNwSvc,
			Value: fmt.Sprintf("vlan://%s/ext-vlan?forwarder=forwarder-vlan", common.GetAppNsName(nseName, cr)),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_DIAL_TIMEOUT" ||
			e.Name == "NSM_REQUEST_TIMEOUT" {
			env = append(env, e)
		}
	}
	return env
}

func (l *LoadBalancer) getFeEnvVars(con *corev1.Container, cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  feEnvConfig,
			Value: common.GetConfigMapName(cr),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "NFE_NAMESPACE" ||
			e.Name == "NFE_GATEWAYS" ||
			e.Name == "NFE_LOG_BIRD" ||
			e.Name == "NFE_ECMP" {
			env = append(env, e)
		}
	}
	return env
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
	dep.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0] = loadBalancerDeploymentName
	dep.Spec.Template.Spec.ServiceAccountName = getServiceAccountName(cr)
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, lbImage, common.Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = common.PullPolicy
	dep.Spec.Template.Spec.Containers[0].LivenessProbe = common.GetLivenessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].ReadinessProbe = common.GetReadinessProbe(cr)
	dep.Spec.Template.Spec.Containers[0].Env = l.getLbEnvVars(&dep.Spec.Template.Spec.Containers[0], cr)
	dep.Spec.Template.Spec.Containers[1].Env = l.getNscEnvVars(&dep.Spec.Template.Spec.Containers[1], cr)
	dep.Spec.Template.Spec.Containers[2].Env = l.getFeEnvVars(&dep.Spec.Template.Spec.Containers[2], cr)

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

func (l *LoadBalancer) getAction(e *common.Executor, cr *meridiov1alpha1.Trench) (common.Action, error) {
	var action common.Action
	err := l.getCurrentStatus(e.Ctx, cr, e.Client)
	if err != nil {
		return nil, err
	}
	if l.currentStatus == nil {
		err = l.getDesiredStatus(cr)
		if err != nil {
			return nil, err
		}
		e.Log.Info("load-balancer", "add action", "create")
		action = common.NewCreateAction(l.desiredStatus, "create load-balncer deployment")
	} else {
		l.getReconciledDesiredStatus(l.currentStatus, cr)
		if !equality.Semantic.DeepEqual(l.desiredStatus, l.currentStatus) {
			e.Log.Info("load-balancer", "add action", "update")
			action = common.NewUpdateAction(l.desiredStatus, "update load-balncer deployment")
		}
	}
	return action, nil
}
