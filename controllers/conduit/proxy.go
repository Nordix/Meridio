package conduit

import (
	"fmt"
	"strconv"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imageProxy = "proxy"
)

type Proxy struct {
	trench  *meridiov1alpha1.Trench
	conduit *meridiov1alpha1.Conduit
	model   *appsv1.DaemonSet
	exec    *common.Executor
}

func NewProxy(e *common.Executor, t *meridiov1alpha1.Trench, c *meridiov1alpha1.Conduit) (*Proxy, error) {
	l := &Proxy{
		trench:  t.DeepCopy(),
		conduit: c.DeepCopy(),
		exec:    e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *Proxy) getEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	env := []corev1.EnvVar{
		{
			Name:  "NSM_SERVICE_NAME",
			Value: common.ProxyNtwkSvcNsName(i.conduit),
		},
		{
			Name:  "NSM_IPAM_SERVICE",
			Value: common.IPAMServiceWithPort(i.trench),
		},
		{
			Name: "NSM_NETWORK_SERVICE_NAME",
			Value: common.LoadBalancerNsName(i.conduit.ObjectMeta.Name,
				i.trench.ObjectMeta.Name,
				i.conduit.ObjectMeta.Namespace),
		},
		{
			Name:  "NSM_IP_FAMILY",
			Value: common.GetIPFamily(i.trench),
		},
		{
			Name:  "NSM_TRENCH",
			Value: i.trench.GetName(),
		},
		{
			Name:  "NSM_CONDUIT",
			Value: i.conduit.ObjectMeta.GetName(),
		},
		{
			Name:  "NSM_NSP_SERVICE_NAME",
			Value: common.GetPrefixedName(common.NspSvcName),
		},
		{
			Name:  "NSM_NSP_SERVICE_PORT",
			Value: strconv.Itoa(common.NspTargetPort),
		},
		{
			Name:  "NSM_NAMESPACE",
			Value: i.conduit.ObjectMeta.Namespace,
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_HOST" ||
			e.Name == "NSM_CONNECT_TO" ||
			e.Name == "NSM_MAX_TOKEN_LIFETIME" ||
			e.Name == "NSM_LOG_LEVEL" {
			env = append(env, e)
		}
	}

	return env
}

func (i *Proxy) insertParameters(init *appsv1.DaemonSet) *appsv1.DaemonSet {
	// if status proxy daemonset parameters are specified in the cr, use those
	// else use the default parameters
	proxyDeploymentName := common.ProxyDeploymentName(i.trench)
	ds := init.DeepCopy()
	ds.ObjectMeta.Name = proxyDeploymentName
	ds.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ds.ObjectMeta.Labels["app"] = proxyDeploymentName
	ds.Spec.Selector.MatchLabels["app"] = proxyDeploymentName
	ds.Spec.Template.ObjectMeta.Labels["app"] = proxyDeploymentName

	ds.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()

	// init container
	if ds.Spec.Template.Spec.InitContainers[0].Image == "" {
		ds.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, common.BusyboxImage, common.BusyboxTag)
	}
	ds.Spec.Template.Spec.InitContainers[0].Args = []string{
		"-c",
		common.GetProxySysCtl(i.trench),
	}

	for x, container := range ds.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "proxy":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageProxy, common.Tag)
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
			container.Env = i.getEnvVars(container.Env)
		default:
			i.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		ds.Spec.Template.Spec.Containers[x] = container
	}
	return ds
}

func (i *Proxy) getModel() error {
	model, err := common.GetDaemonsetModel("deployment/proxy.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *Proxy) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.ProxyDeploymentName(i.trench),
	}
}

func (i *Proxy) getDesiredStatus() *appsv1.DaemonSet {
	return i.insertParameters(i.model)
}

// getReconciledDesiredStatus gets the desired status of proxy daemonset after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *Proxy) getReconciledDesiredStatus(cd *appsv1.DaemonSet) *appsv1.DaemonSet {
	template := cd.DeepCopy()
	template.Spec.Template.Spec.Containers = i.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.InitContainers = i.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Volumes = i.model.Spec.Template.Spec.Volumes
	return i.insertParameters(template)
}

func (i *Proxy) getCurrentStatus() (*appsv1.DaemonSet, error) {
	currentStatus := &appsv1.DaemonSet{}
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

func (i *Proxy) getAction() error {
	cs, err := i.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds.Spec, cs.Spec) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
