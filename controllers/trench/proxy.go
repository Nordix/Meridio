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

	busyboxImage = "busybox"
	busyboxTag   = "1.29"

	proxyEnvConfig        = "NSM_CONFIG_MAP_NAME"
	proxyEnvService       = "NSM_SERVICE_NAME"
	proxyEnvSubnetPools   = "NSM_SUBNET_POOLS"
	proxyEnvSubnetLengths = "NSM_SUBNET_PREFIX_LENGTHS"
	proxyEnvIpam          = "NSM_IPAM_SERVICE"
	proxyEnvLb            = "NSM_NETWORK_SERVICE_NAME"
)

type Proxy struct {
	currentStatus *appsv1.DaemonSet
	desiredStatus *appsv1.DaemonSet
}

func (i *Proxy) getEnvVars(ds *appsv1.DaemonSet, cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	allEnv := ds.Spec.Template.Spec.Containers[0].Env
	env := []corev1.EnvVar{
		{
			Name:  proxyEnvConfig,
			Value: common.ConfigMapName(cr),
		},
		{
			Name:  proxyEnvSubnetPools,
			Value: common.GetSubnetPool(cr),
		},
		{
			Name:  proxyEnvSubnetLengths,
			Value: common.GetPrefixLength(cr),
		},
		{
			Name:  proxyEnvService,
			Value: common.ProxyNtwkSvcNsName(cr),
		},
		{
			Name:  proxyEnvIpam,
			Value: common.IPAMServiceWithPort(cr),
		},
		{
			Name:  proxyEnvLb,
			Value: common.LoadBalancerNsName(cr),
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

func (i *Proxy) insertParamters(ds *appsv1.DaemonSet, cr *meridiov1alpha1.Trench) *appsv1.DaemonSet {
	// if status proxy daemonset parameters are specified in the cr, use those
	// else use the default parameters
	proxyDeploymentName := common.ProxyDeploymentName(cr)
	ds.ObjectMeta.Name = proxyDeploymentName
	ds.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	ds.ObjectMeta.Labels["app"] = proxyDeploymentName
	ds.Spec.Selector.MatchLabels["app"] = proxyDeploymentName
	ds.Spec.Template.ObjectMeta.Labels["app"] = proxyDeploymentName
	ds.Spec.Template.Spec.ServiceAccountName = common.ServiceAccountName(cr)
	// init container
	ds.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, busyboxImage, busyboxTag)
	// proxy container
	ds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageProxy, common.Tag)
	ds.Spec.Template.Spec.Containers[0].ImagePullPolicy = common.PullPolicy
	ds.Spec.Template.Spec.Containers[0].LivenessProbe = common.GetLivenessProbe(cr)
	ds.Spec.Template.Spec.Containers[0].ReadinessProbe = common.GetLivenessProbe(cr)
	ds.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(ds, cr)
	return ds
}

func (i *Proxy) getModel() (*appsv1.DaemonSet, error) {
	return common.GetDaemonsetModel("deployment/proxy.yaml")
}

func (i *Proxy) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      common.ProxyDeploymentName(cr),
	}
}

func (i *Proxy) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	proxy, err := i.getModel()
	if err != nil {
		return err
	}
	i.desiredStatus = i.insertParamters(proxy, cr)
	return nil
}

// getReconciledDesiredStatus gets the desired status of proxy daemonset after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *Proxy) getReconciledDesiredStatus(cd *appsv1.DaemonSet, cr *meridiov1alpha1.Trench) {
	i.desiredStatus = i.insertParamters(cd, cr)
}

func (i *Proxy) getCurrentStatus(e *common.Executor, cr *meridiov1alpha1.Trench) error {
	currentStatus := &appsv1.DaemonSet{}
	selector := i.getSelector(cr)
	err := e.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	i.currentStatus = currentStatus.DeepCopy()
	return nil
}

func (i *Proxy) getAction(e *common.Executor, cr *meridiov1alpha1.Trench) (common.Action, error) {
	var action common.Action
	err := i.getCurrentStatus(e, cr)
	if err != nil {
		return action, err
	}
	if i.currentStatus == nil {
		err := i.getDesiredStatus(cr)
		if err != nil {
			return action, err
		}
		e.LogInfo("add action: create proxy daemonset")
		action = common.NewCreateAction(i.desiredStatus, "create proxy daemonset")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.LogInfo("add action: update proxy daemonset")
			action = common.NewUpdateAction(i.desiredStatus, "update proxy daemonset")
		}
	}
	return action, nil
}
