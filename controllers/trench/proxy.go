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
	imageProxy = "proxy"

	proxyEnvConfig        = "NSM_CONFIG_MAP_NAME"
	proxyEnvService       = "NSM_SERVICE_NAME"
	proxyEnvSubnetPools   = "NSM_SUBNET_POOLS"
	proxyEnvSubnetLengths = "NSM_SUBNET_PREFIX_LENGTHS"
	proxyEnvIpam          = "NSM_IPAM_SERVICE"
	proxyEnvLb            = "NSM_NETWORK_SERVICE_NAME"
)

type Proxy struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.DaemonSet
	exec   *common.Executor
}

func NewProxy(e *common.Executor, t *meridiov1alpha1.Trench) (*Proxy, error) {
	l := &Proxy{
		trench: t.DeepCopy(),
		exec:   e,
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
			Name:  proxyEnvConfig,
			Value: common.ConfigMapName(i.trench),
		},
		{
			Name:  proxyEnvSubnetPools,
			Value: common.GetSubnetPool(i.trench),
		},
		{
			Name:  proxyEnvSubnetLengths,
			Value: common.GetPrefixLength(i.trench),
		},
		{
			Name:  proxyEnvService,
			Value: common.ProxyNtwkSvcNsName(i.trench),
		},
		{
			Name:  proxyEnvIpam,
			Value: common.IPAMServiceWithPort(i.trench),
		},
		{
			Name:  proxyEnvLb,
			Value: common.LoadBalancerNsName(i.trench),
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSM_NAME" ||
			e.Name == "NSM_HOST" ||
			e.Name == "NSM_NAMESPACE" ||
			e.Name == "NSM_CONNECT_TO" {
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
	ds.Spec.Template.Spec.ServiceAccountName = common.ServiceAccountName(i.trench)

	ds.Spec.Template.Spec.ImagePullSecrets = common.GetImagePullSecrets()

	// init container
	if ds.Spec.Template.Spec.InitContainers[0].Image == "" {
		ds.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, common.BusyboxImage, common.BusyboxTag)
	}
	ds.Spec.Template.Spec.InitContainers[0].Args = []string{
		"-c",
		common.GetProxySysCtl(i.trench),
	}
	// proxy container
	if ds.Spec.Template.Spec.Containers[0].Image == "" {
		ds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageProxy, common.Tag)
	}
	ds.Spec.Template.Spec.Containers[0].LivenessProbe = common.GetLivenessProbe(i.trench)
	ds.Spec.Template.Spec.Containers[0].ReadinessProbe = common.GetLivenessProbe(i.trench)
	ds.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(ds.Spec.Template.Spec.Containers[0].Env)
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
	return i.insertParameters(cd)
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

func (i *Proxy) getAction() ([]common.Action, error) {
	var action []common.Action
	cs, err := i.getCurrentStatus()
	if err != nil {
		return action, err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		action = append(action, i.exec.NewCreateAction(ds))
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			action = append(action, i.exec.NewUpdateAction(ds))
		}
	}
	return action, nil
}
