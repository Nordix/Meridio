package attractor

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lbImage      = "load-balancer"
	lbEnvConfig  = "NSM_CONFIG_MAP_NAME"
	lbEnvService = "NSM_SERVICE_NAME"
	lbEnvNsp     = "NSM_NSP_SERVICE"
	nscEnvNwSvc  = "NSM_NETWORK_SERVICES"
	feEnvConfig  = "NFE_CONFIG_MAP_NAME"
)

type LoadBalancer struct {
	model     *appsv1.Deployment
	trench    *meridiov1alpha1.Trench
	attractor *meridiov1alpha1.Attractor
}

func (l *LoadBalancer) getModel() error {
	model, err := common.GetDeploymentModel("deployment/lb-fe.yaml")
	if err != nil {
		return err
	}
	l.model = model
	return nil
}

func NewLoadBalancer(e *common.Executor, attr *meridiov1alpha1.Attractor, t *meridiov1alpha1.Trench) (*LoadBalancer, error) {
	l := &LoadBalancer{
		attractor: attr,
		trench:    t.DeepCopy(),
	}

	// get model of load balancer
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *LoadBalancer) getLbEnvVars(con corev1.Container) []corev1.EnvVar {
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  lbEnvConfig,
			Value: common.ConfigMapName(l.trench),
		},
		{
			Name:  lbEnvService,
			Value: common.LoadBalancerNsName(l.trench),
		},
		{
			Name:  lbEnvNsp,
			Value: common.NSPServiceWithPort(l.trench),
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

func (l *LoadBalancer) getNscEnvVars(con corev1.Container) []corev1.EnvVar {
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  nscEnvNwSvc,
			Value: fmt.Sprintf("vlan://%s/ext-vlan?forwarder=forwarder-vlan", common.NSENsName(l.attractor)),
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

func (l *LoadBalancer) getFeEnvVars(con corev1.Container) []corev1.EnvVar {
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  feEnvConfig,
			Value: common.ConfigMapName(l.trench),
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

func (l *LoadBalancer) insertParamters(dep *appsv1.Deployment) *appsv1.Deployment {
	// if status lb-fe deployment parameters are specified in the cr, use those
	// else use the default parameters
	loadBalancerDeploymentName := common.LoadBalancerDeploymentName(l.trench)
	ret := dep.DeepCopy()
	ret.ObjectMeta.Name = loadBalancerDeploymentName
	ret.ObjectMeta.Namespace = l.trench.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = loadBalancerDeploymentName
	ret.Spec.Selector.MatchLabels["app"] = loadBalancerDeploymentName
	ret.Spec.Template.ObjectMeta.Labels["app"] = loadBalancerDeploymentName
	ret.Spec.Replicas = l.attractor.Spec.Replicas
	ret.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0] = loadBalancerDeploymentName
	ret.Spec.Template.Spec.ServiceAccountName = common.ServiceAccountName(l.trench)
	ret.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, lbImage, common.Tag)
	ret.Spec.Template.Spec.Containers[0].ImagePullPolicy = common.PullPolicy
	ret.Spec.Template.Spec.Containers[0].LivenessProbe = common.GetLivenessProbe(l.trench)
	ret.Spec.Template.Spec.Containers[0].ReadinessProbe = common.GetReadinessProbe(l.trench)
	ret.Spec.Template.Spec.Containers[0].Env = l.getLbEnvVars(ret.Spec.Template.Spec.Containers[0])
	ret.Spec.Template.Spec.Containers[1].Env = l.getNscEnvVars(ret.Spec.Template.Spec.Containers[1])
	ret.Spec.Template.Spec.Containers[2].Env = l.getFeEnvVars(ret.Spec.Template.Spec.Containers[2])

	return ret
}

func (l *LoadBalancer) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: l.trench.ObjectMeta.Namespace,
		Name:      common.LoadBalancerDeploymentName(l.trench),
	}
}

func (l *LoadBalancer) getCurrentStatus(e *common.Executor) (*appsv1.Deployment, error) {
	currentState := &appsv1.Deployment{}
	err := e.GetObject(l.getSelector(), currentState)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentState, nil
}

func (l *LoadBalancer) getDesiredStatus() *appsv1.Deployment {
	return l.insertParamters(l.model)
}

// getReconciledDesiredStatus gets the desired status of lb-fe deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *LoadBalancer) getReconciledDesiredStatus(lb *appsv1.Deployment) *appsv1.Deployment {
	return i.insertParamters(lb)
}

func (l *LoadBalancer) getAction(e *common.Executor) (common.Action, error) {
	elem := common.LoadBalancerDeploymentName(l.trench)
	var action common.Action
	cs, err := l.getCurrentStatus(e)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		ds := l.getDesiredStatus()
		if err != nil {
			return nil, err
		}
		e.LogInfo(fmt.Sprintf("add action: create %s", elem))
		action = common.NewCreateAction(ds, fmt.Sprintf("create %s", elem))
	} else {
		ds := l.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			e.LogInfo(fmt.Sprintf("add action: update %s", elem))
			action = common.NewUpdateAction(ds, fmt.Sprintf("update %s", elem))
		}
	}
	return action, nil
}
