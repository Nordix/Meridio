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
	lbImage = "load-balancer"
	feImage = "frontend"
)

var lbFeDeploymentName string

type LoadBalancer struct {
	model     *appsv1.Deployment
	trench    *meridiov1alpha1.Trench
	attractor *meridiov1alpha1.Attractor
	exec      *common.Executor
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
		exec:      e,
		trench:    t.DeepCopy(),
		attractor: attr.DeepCopy(),
	}
	lbFeDeploymentName = common.LbFeDeploymentName(l.attractor)
	// get model of load balancer
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *LoadBalancer) getLbEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name: "NSM_SERVICE_NAME",
			Value: common.LoadBalancerNsName(l.attractor.Spec.Composites[0],
				l.trench.ObjectMeta.Name,
				l.attractor.ObjectMeta.Namespace),
		},
		{
			Name:  "NSM_CONDUIT_NAME",
			Value: l.attractor.Spec.Composites[0],
		},
		{
			Name:  "NSM_TRENCH_NAME",
			Value: l.trench.ObjectMeta.Name,
		},
		{
			Name:  "NSM_NSP_SERVICE",
			Value: common.NSPServiceWithPort(l.trench),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_MAX_TOKEN_LIFETIME" ||
			e.Name == "NSM_LOG_LEVEL" {
			env = append(env, e)
		}
	}
	return env
}

func (l *LoadBalancer) getNscEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "NSM_NETWORK_SERVICES",
			Value: fmt.Sprintf("kernel://%s/%s", common.VlanNtwkSvcName(l.trench), common.GetExternalInterfaceName(l.attractor)),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_DIAL_TIMEOUT" ||
			e.Name == "NSM_REQUEST_TIMEOUT" ||
			e.Name == "NSM_MAX_TOKEN_LIFETIME" ||
			e.Name == "NSM_LOG_LEVEL" {
			env = append(env, e)
		}
	}
	return env
}

func (l *LoadBalancer) getFeEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "NFE_CONFIG_MAP_NAME",
			Value: common.ConfigMapName(l.trench),
		},
		{
			Name:  "NFE_NSP_SERVICE",
			Value: common.NSPServiceWithPort(l.trench),
		},
		{
			Name:  "NFE_TRENCH_NAME",
			Value: l.trench.ObjectMeta.Name,
		},
		{
			Name:  "NFE_ATTRACTOR_NAME",
			Value: l.attractor.ObjectMeta.Name,
		},
		{
			Name:  "NFE_NAMESPACE",
			Value: l.attractor.ObjectMeta.Namespace,
		},
		{
			Name:  "NFE_EXTERNAL_INTERFACE",
			Value: common.GetExternalInterfaceName(l.attractor),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NFE_LOG_BIRD" ||
			e.Name == "NFE_ECMP" ||
			e.Name == "NFE_LOG_LEVEL" {
			env = append(env, e)
		}
	}
	return env
}

func (l *LoadBalancer) insertParameters(dep *appsv1.Deployment) *appsv1.Deployment {
	// if status lb-fe deployment parameters are specified in the cr, use those
	// else use the default parameters
	ret := dep.DeepCopy()
	ret.ObjectMeta.Name = lbFeDeploymentName
	ret.ObjectMeta.Namespace = l.trench.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = lbFeDeploymentName
	ret.Spec.Selector.MatchLabels["app"] = lbFeDeploymentName

	ret.Spec.Replicas = l.attractor.Spec.Replicas

	ret.Spec.Template.ObjectMeta.Labels["app"] = lbFeDeploymentName
	ret.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0] = lbFeDeploymentName
	ret.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()

	if ret.Spec.Template.Spec.InitContainers[0].Image == "" {
		ret.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, common.BusyboxImage, common.BusyboxTag)
		ret.Spec.Template.Spec.InitContainers[0].ImagePullPolicy = corev1.PullIfNotPresent
	}
	ret.Spec.Template.Spec.InitContainers[0].Args = []string{
		"-c",
		common.GetLoadBalancerSysCtl(l.trench),
	}

	for i, container := range ret.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "load-balancer":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, lbImage, common.Tag)
				container.ImagePullPolicy = corev1.PullAlways
			}
			if container.StartupProbe == nil {
				container.StartupProbe = common.GetProbe(common.StartUpTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			if container.ReadinessProbe == nil {
				container.ReadinessProbe = common.GetProbe(common.ReadinessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", "Readiness"))
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.LivenessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			container.Env = l.getLbEnvVars(container.Env)
		case "nsc":
			if container.Image == "" {
				container.Image = "registry.nordix.org/cloud-native/nsm/cmd-nsc:v1.2.0-rc.1"
				container.ImagePullPolicy = corev1.PullAlways
			}
			container.Env = l.getNscEnvVars(container.Env)
		case "fe":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, feImage, common.Tag)
				container.ImagePullPolicy = corev1.PullAlways
			}
			if container.StartupProbe == nil {
				container.StartupProbe = common.GetProbe(common.StartUpTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			if container.ReadinessProbe == nil {
				container.ReadinessProbe = common.GetProbe(common.ReadinessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", "Readiness"))
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.LivenessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			container.Env = l.getFeEnvVars(container.Env)
		default:
			l.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		ret.Spec.Template.Spec.Containers[i] = container
	}

	return ret
}

func (l *LoadBalancer) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: l.trench.ObjectMeta.Namespace,
		Name:      lbFeDeploymentName,
	}
}

func (l *LoadBalancer) getCurrentStatus() (*appsv1.Deployment, error) {
	currentState := &appsv1.Deployment{}
	err := l.exec.GetObject(l.getSelector(), currentState)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentState, nil
}

func (l *LoadBalancer) getDesiredStatus() *appsv1.Deployment {
	return l.insertParameters(l.model)
}

// getReconciledDesiredStatus gets the desired status of lb-fe deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *LoadBalancer) getReconciledDesiredStatus(lb *appsv1.Deployment) *appsv1.Deployment {
	template := lb.DeepCopy()
	template.Spec.Template.Spec.InitContainers = i.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Containers = i.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.Volumes = i.model.Spec.Template.Spec.Volumes
	return i.insertParameters(template)
}

func (l *LoadBalancer) getAction() error {
	cs, err := l.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := l.getDesiredStatus()
		if err != nil {
			return err
		}
		l.exec.AddCreateAction(ds)
	} else {
		ds := l.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds.Spec, cs.Spec) {
			l.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
