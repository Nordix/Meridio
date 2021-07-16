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
	nameProxy             = "proxy"
	imageProxy            = "proxy"
	proxyEnvConfig        = "NSM_CONFIG_MAP_NAME"
	proxyEnvService       = "NSM_SERVICE_NAME"
	proxyEnvSubnetPools   = "NSM_SUBNET_POOLS"
	proxyEnvSubnetLengths = "NSM_SUBNET_PREFIX_LENGTHS"
	proxyEnvIpam          = "NSM_IPAM_SERVICE"
	proxyEnvLb            = "NSM_NETWORK_SERVICE_NAME"
)

func getProxyDeploymentName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s-%s", nameProxy, cr.ObjectMeta.Name)
}

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
			Value: getConfigMapName(cr),
		},
		{
			Name:  proxyEnvSubnetPools,
			Value: getSubnetPool(cr),
		},
		{
			Name:  proxyEnvSubnetLengths,
			Value: getPrefixLength(cr),
		},
		{
			Name:  proxyEnvService,
			Value: getProxyNsName(cr),
		},
		{
			Name:  proxyEnvIpam,
			Value: getIPAMService(cr),
		},
		{
			Name:  proxyEnvLb,
			Value: getLoadBalancerNsName(cr),
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
	proxyDeploymentName := getProxyDeploymentName(cr)
	ds.ObjectMeta.Name = proxyDeploymentName
	ds.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	ds.ObjectMeta.Labels["app"] = proxyDeploymentName
	ds.Spec.Selector.MatchLabels["app"] = proxyDeploymentName
	ds.Spec.Template.ObjectMeta.Labels["app"] = proxyDeploymentName
	ds.Spec.Template.Spec.ServiceAccountName = getServiceAccountName(cr)
	// init container
	ds.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, Organization, busyboxImage, busyboxTag)
	// proxy container
	ds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, Organization, imageProxy, Tag)
	ds.Spec.Template.Spec.Containers[0].ImagePullPolicy = PullPolicy
	ds.Spec.Template.Spec.Containers[0].LivenessProbe = GetLivenessProbe(cr)
	ds.Spec.Template.Spec.Containers[0].ReadinessProbe = GetReadinessProbe(cr)
	ds.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(ds, cr)
	return ds
}

func (i *Proxy) getModel() (*appsv1.DaemonSet, error) {
	return getDaemonsetModel("deployment/proxy.yaml")
}

func (i *Proxy) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getProxyDeploymentName(cr),
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

func (i *Proxy) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
	currentStatus := &appsv1.DaemonSet{}
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

func (i *Proxy) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
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
		e.log.Info("proxy daemonset", "add action", "create")
		action = newCreateAction(i.desiredStatus, "create proxy daemonset")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.log.Info("proxy daemonset", "add action", "update")
			action = newUpdateAction(i.desiredStatus, "update proxy daemonset")
		}
	}
	return action, nil
}
