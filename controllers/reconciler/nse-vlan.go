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
	nseName        = "nse-vlan"
	nseImage       = "nse-vlan"
	nseEnvItf      = "NSE_VLAN_BASE_IFNAME"
	nseEnvID       = "NSE_VLAN_ID"
	nseEnvSerive   = "NSE_SERVICE_NAME"
	nseEnvPrefixV4 = "NSE_CIDR_PREFIX"
	nseEnvPrefixV6 = "NSE_IPV6_PREFIX"
)

func getNSEVLANDeploymentName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s-%s", nseName, cr.ObjectMeta.Name)
}

type NseDeployment struct {
	currentStatus *appsv1.Deployment
	desiredStatus *appsv1.Deployment
}

func (i *NseDeployment) getEnvVars(dp *appsv1.Deployment, cr *meridiov1alpha1.Trench) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	allEnv := dp.Spec.Template.Spec.Containers[0].Env
	env := []corev1.EnvVar{
		{
			Name:  nseEnvItf,
			Value: vlanItf,
		},
		{
			Name:  nseEnvID,
			Value: vlanID,
		},
		{
			Name:  nseEnvSerive,
			Value: getVlanNsName(cr),
		},
		{
			Name:  nseEnvPrefixV4,
			Value: vlanPrefixV4,
		},
		{
			Name:  nseEnvPrefixV6,
			Value: vlanPrefixV6,
		},
	}

	for _, e := range allEnv {
		// append all hard coded envVars
		if e.Name == "SPIFFE_ENDPOINT_SOCKET" ||
			e.Name == "NSE_NAME" ||
			e.Name == "NSE_CONNECT_TO" ||
			e.Name == "NSE_POINT2POINT" {
			env = append(env, e)
		}
	}
	return env
}

func (i *NseDeployment) insertParamters(dep *appsv1.Deployment, cr *meridiov1alpha1.Trench) *appsv1.Deployment {
	// if status nse deployment parameters are specified in the cr, use those
	// else use the default parameters
	nseVLANDeploymentName := getNSEVLANDeploymentName(cr)
	dep.ObjectMeta.Name = nseVLANDeploymentName
	dep.ObjectMeta.Namespace = cr.ObjectMeta.Namespace
	dep.ObjectMeta.Labels["app"] = nseVLANDeploymentName
	dep.Spec.Selector.MatchLabels["app"] = nseVLANDeploymentName
	dep.Spec.Template.ObjectMeta.Labels["app"] = nseVLANDeploymentName
	dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", Registry, OrganizationNsm, nseImage, Tag)
	dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = PullPolicy
	dep.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(dep, cr)
	return dep
}

func (i *NseDeployment) getModel() (*appsv1.Deployment, error) {
	return getDeploymentModel("deployment/nse-vlan.yaml")
}

func (i *NseDeployment) getSelector(cr *meridiov1alpha1.Trench) client.ObjectKey {
	return client.ObjectKey{
		Namespace: cr.ObjectMeta.Namespace,
		Name:      getNSEVLANDeploymentName(cr),
	}
}

func (i *NseDeployment) getDesiredStatus(cr *meridiov1alpha1.Trench) error {
	NseDeployment, err := i.getModel()
	if err != nil {
		return err
	}
	i.desiredStatus = i.insertParamters(NseDeployment, cr)
	return nil
}

// getReconciledDesiredStatus gets the desired status of nse deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NseDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment, cr *meridiov1alpha1.Trench) {
	i.desiredStatus = i.insertParamters(cd, cr)
}

func (i *NseDeployment) getCurrentStatus(ctx context.Context, cr *meridiov1alpha1.Trench, client client.Client) error {
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

func (i *NseDeployment) getAction(e *Executor, cr *meridiov1alpha1.Trench) (Action, error) {
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
		e.log.Info("nse deployment", "add action", "create")
		action = newCreateAction(i.desiredStatus, "create nse deployment")
	} else {
		i.getReconciledDesiredStatus(i.currentStatus, cr)
		if !equality.Semantic.DeepEqual(i.desiredStatus, i.currentStatus) {
			e.log.Info("nse deployment", "add action", "update")
			action = newUpdateAction(i.desiredStatus, "update nse deployment")
		}
	}
	return action, nil
}
