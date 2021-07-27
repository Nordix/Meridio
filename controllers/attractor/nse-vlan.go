package attractor

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
	nseImage       = "nse-vlan"
	nseEnvItf      = "NSE_VLAN_BASE_IFNAME"
	nseEnvID       = "NSE_VLAN_ID"
	nseEnvSerive   = "NSE_SERVICE_NAME"
	nseEnvPrefixV4 = "NSE_CIDR_PREFIX"
	nseEnvPrefixV6 = "NSE_IPV6_PREFIX"

	vlanPrefixIPv4 = "169.254.100.0/24"
	vlanPrefixIPv6 = "100:100::/64"
)

type NseDeployment struct {
	model     *appsv1.Deployment
	attractor *meridiov1alpha1.Attractor
}

func NewNSE(e *common.Executor, attr *meridiov1alpha1.Attractor) (*NseDeployment, error) {
	nse := &NseDeployment{
		attractor: attr,
	}
	err := nse.getModel()
	if err != nil {
		return nil, err
	}
	return nse, nil
}

func (i *NseDeployment) getEnvVars(con corev1.Container) []corev1.EnvVar {
	// if envVars are set in the cr, use the values
	// else return default envVars
	allEnv := con.Env
	env := []corev1.EnvVar{
		{
			Name:  nseEnvItf,
			Value: i.attractor.Spec.VlanInterface,
		},
		{
			Name:  nseEnvID,
			Value: fmt.Sprint(i.attractor.Spec.VlanID),
		},
		{
			Name:  nseEnvSerive,
			Value: common.NSENsName(i.attractor),
		},
		{
			Name:  nseEnvPrefixV4,
			Value: vlanPrefixIPv4,
		},
		{
			Name:  nseEnvPrefixV6,
			Value: vlanPrefixIPv6,
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

func (i *NseDeployment) insertParamters(dep *appsv1.Deployment) *appsv1.Deployment {
	ret := dep.DeepCopy()
	nseVLANDeploymentName := common.NSEDeploymentName(i.attractor)
	ret.ObjectMeta.Name = nseVLANDeploymentName
	ret.ObjectMeta.Namespace = i.attractor.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = nseVLANDeploymentName
	ret.Spec.Replicas = i.attractor.Spec.NSEReplicas
	ret.Spec.Selector.MatchLabels["app"] = nseVLANDeploymentName
	ret.Spec.Template.ObjectMeta.Labels["app"] = nseVLANDeploymentName
	ret.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.OrganizationNsm, nseImage, common.Tag)
	ret.Spec.Template.Spec.Containers[0].ImagePullPolicy = common.PullPolicy
	ret.Spec.Template.Spec.Containers[0].Env = i.getEnvVars(ret.Spec.Template.Spec.Containers[0])
	return ret
}

func (i *NseDeployment) getModel() error {
	model, err := common.GetDeploymentModel("deployment/nse-vlan.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *NseDeployment) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.attractor.ObjectMeta.Namespace,
		Name:      common.NSEDeploymentName(i.attractor),
	}
}

func (i *NseDeployment) getDesiredStatus() *appsv1.Deployment {
	return i.insertParamters(i.model)

}

// getReconciledDesiredStatus gets the desired status of nse deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NseDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment) *appsv1.Deployment {
	return i.insertParamters(cd)
}

func (i *NseDeployment) getCurrentStatus(e *common.Executor) (*appsv1.Deployment, error) {
	currentStatus := &appsv1.Deployment{}
	selector := i.getSelector()
	err := e.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return currentStatus, nil
}

func (i *NseDeployment) getAction(e *common.Executor) ([]common.Action, error) {
	var actions []common.Action
	// update attractor status
	cs, err := i.getCurrentStatus(e)
	if err != nil {
		return actions, err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		e.LogInfo("add action: create nse deployment")
		actions = append(actions, common.NewCreateAction(ds, "create nse deployment"))
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds, cs) {
			e.LogInfo("add action: update nse deployment")
			actions = append(actions, common.NewUpdateAction(ds, "update nse deployment"))
		}
	}
	i.attractor.Status.Vlan = meridiov1alpha1.DeploymentStatus.Deployed
	return actions, nil
}
