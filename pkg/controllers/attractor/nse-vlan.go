/*
Copyright (c) 2021-2022 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package attractor

import (
	"fmt"

	meridiov1 "github.com/nordix/meridio/api/v1"
	common "github.com/nordix/meridio/pkg/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nseImage    = "cmd-nse-remote-vlan"
	nseImageTag = "v1.13.0"
)

type NseDeployment struct {
	model     *appsv1.Deployment
	attractor *meridiov1.Attractor
	exec      *common.Executor
	trench    *meridiov1.Trench
}

func NewNSE(e *common.Executor, attr *meridiov1.Attractor, t *meridiov1.Trench) (*NseDeployment, error) {
	nse := &NseDeployment{
		attractor: attr,
		exec:      e,
		trench:    t,
	}
	err := nse.getModel()
	if err != nil {
		return nil, err
	}
	return nse, nil
}

func (i *NseDeployment) getEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	operatorEnv := []corev1.EnvVar{
		{Name: "NSM_SERVICES", Value: fmt.Sprintf("%s { vlan: %d; via: %s }",
			common.VlanNtwkSvcName(i.attractor, i.trench),
			*i.attractor.Spec.Interface.NSMVlan.VlanID,
			i.attractor.Spec.Interface.NSMVlan.BaseInterface)},
		{Name: "NSM_CONNECT_TO", Value: common.GetNSMRegistryService()},
		{Name: "NSM_CIDR_PREFIX", Value: fmt.Sprintf("%v,%v", i.attractor.Spec.Interface.PrefixIPv4, i.attractor.Spec.Interface.PrefixIPv6)},
		{Name: "NSM_LOG_LEVEL", Value: common.GetLogLevel()},
		{Name: "NSM_LISTEN_ON", Value: fmt.Sprintf("tcp://:%v", common.VlanNsePort)},
	}
	return common.CompileEnvironmentVariables(allEnv, operatorEnv)
}

func (i *NseDeployment) insertParameters(dep *appsv1.Deployment) *appsv1.Deployment {
	ret := dep.DeepCopy()
	nseVLANDeploymentName := common.NSEDeploymentName(i.attractor)
	ret.ObjectMeta.Name = nseVLANDeploymentName
	ret.ObjectMeta.Namespace = i.attractor.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = nseVLANDeploymentName
	ret.Spec.Selector.MatchLabels["app"] = nseVLANDeploymentName
	ret.Spec.Template.ObjectMeta.Labels["app"] = nseVLANDeploymentName

	imagePullSecrets := common.GetImagePullSecrets()
	if len(imagePullSecrets) > 0 {
		ret.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}

	// check resource requirement annotation update, and save annotation into deployment for visibility
	oa, _ := common.GetResourceRequirementAnnotation(&dep.ObjectMeta)
	if na, _ := common.GetResourceRequirementAnnotation(&i.attractor.ObjectMeta); na != oa {
		common.SetResourceRequirementAnnotation(&i.attractor.ObjectMeta, &ret.ObjectMeta)
	}

	for x, container := range ret.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "nse-vlan":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.OrganizationNsm, nseImage, nseImageTag)
				container.ImagePullPolicy = corev1.PullAlways
			}
			if container.StartupProbe == nil {
				container.StartupProbe = common.GetProbe(common.StartUpTimer,
					common.GetProbeCommand(true, fmt.Sprintf(":%d", common.VlanNsePort), ""))
			}
			if container.ReadinessProbe == nil {
				container.ReadinessProbe = common.GetProbe(common.ReadinessTimer,
					common.GetProbeCommand(true, fmt.Sprintf(":%d", common.VlanNsePort), ""))
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.LivenessTimer,
					common.GetProbeCommand(true, fmt.Sprintf(":%d", common.VlanNsePort), ""))
			}
			container.Env = i.getEnvVars(container.Env)
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&i.attractor.ObjectMeta, &container); err != nil {
				i.exec.LogInfo(fmt.Sprintf("attractor %s, %v", i.attractor.ObjectMeta.Name, err))
			}
		default:
			i.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		ret.Spec.Template.Spec.Containers[x] = container
	}
	return ret
}

func (i *NseDeployment) getModel() error {
	model, err := common.GetDeploymentModel("deployment/nse-vlan.yaml")
	if err != nil {
		return fmt.Errorf("failed to get deployment model in deployment/nse-vlan.yaml: %w", err)
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
	return i.insertParameters(i.model)

}

// getReconciledDesiredStatus gets the desired status of nse deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *NseDeployment) getReconciledDesiredStatus(cd *appsv1.Deployment) *appsv1.Deployment {
	template := cd.DeepCopy()
	template.Spec.Template.Spec.InitContainers = i.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Containers = i.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.Volumes = i.model.Spec.Template.Spec.Volumes
	return i.insertParameters(template)
}

func (i *NseDeployment) getCurrentStatus() (*appsv1.Deployment, error) {
	currentStatus := &appsv1.Deployment{}
	selector := i.getSelector()
	err := i.exec.GetObject(selector, currentStatus)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get nse-vlan object (%s): %w", selector.String(), err)
	}
	return currentStatus, nil
}

func (i *NseDeployment) getAction() error {
	cs, err := i.getCurrentStatus()
	if err != nil {
		return err
	}
	if cs == nil {
		ds := i.getDesiredStatus()
		if err != nil {
			return err
		}
		i.exec.AddCreateAction(ds)
	} else {
		ds := i.getReconciledDesiredStatus(cs)
		if !equality.Semantic.DeepEqual(ds.Spec, cs.Spec) {
			i.exec.AddUpdateAction(ds)
		}
	}
	return nil
}
