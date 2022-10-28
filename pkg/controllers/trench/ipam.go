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

package trench

import (
	"fmt"
	"strconv"

	meridiov1alpha1 "github.com/nordix/meridio/api/v1alpha1"
	common "github.com/nordix/meridio/pkg/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	imageIpam = "ipam"
)

type IpamStatefulSet struct {
	trench *meridiov1alpha1.Trench
	model  *appsv1.StatefulSet
	exec   *common.Executor
}

func NewIPAM(e *common.Executor, t *meridiov1alpha1.Trench) (*IpamStatefulSet, error) {
	l := &IpamStatefulSet{
		trench: t.DeepCopy(),
		exec:   e,
	}

	// get model
	if err := l.getModel(); err != nil {
		return nil, err
	}
	return l, nil
}

func (i *IpamStatefulSet) getEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	operatorEnv := map[string]string{
		"IPAM_PORT":                       strconv.Itoa(common.IpamPort),
		"IPAM_NAMESPACE":                  i.trench.ObjectMeta.Namespace,
		"IPAM_TRENCH_NAME":                i.trench.ObjectMeta.GetName(),
		"IPAM_NSP_SERVICE":                common.NSPServiceWithPort(i.trench),
		"IPAM_PREFIX_IPV4":                common.SubnetPoolIpv4,
		"IPAM_PREFIX_IPV6":                common.SubnetPoolIpv6,
		"IPAM_CONDUIT_PREFIX_LENGTH_IPV4": common.ConduitPrefixLengthIpv4,
		"IPAM_CONDUIT_PREFIX_LENGTH_IPV6": common.ConduitPrefixLengthIpv6,
		"IPAM_NODE_PREFIX_LENGTH_IPV4":    common.NodePrefixLengthIpv4,
		"IPAM_NODE_PREFIX_LENGTH_IPV6":    common.NodePrefixLengthIpv6,
		"IPAM_IP_FAMILY":                  common.GetIPFamily(i.trench),
		"IPAM_LOG_LEVEL":                  common.GetLogLevel(),
	}
	return common.CompileEnvironmentVariables(allEnv, operatorEnv)
}

func (i *IpamStatefulSet) insertParameters(dep *appsv1.StatefulSet) *appsv1.StatefulSet {
	// if status ipam statefulset parameters are specified in the cr, use those
	// else use the default parameters
	ret := dep.DeepCopy()
	ipamStatefulSetName := common.IPAMStatefulSetName(i.trench)
	ret.ObjectMeta.Name = ipamStatefulSetName
	ret.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = ipamStatefulSetName
	ret.Spec.Selector.MatchLabels["app"] = ipamStatefulSetName
	ret.Spec.Template.ObjectMeta.Labels["app"] = ipamStatefulSetName

	ret.Spec.ServiceName = ipamStatefulSetName

	imagePullSecrets := common.GetImagePullSecrets()
	if len(imagePullSecrets) > 0 {
		ret.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}

	// check resource requirement annotation update, and save annotation into deployment for visibility
	oa, _ := common.GetResourceRequirementAnnotation(&dep.ObjectMeta)
	if na, _ := common.GetResourceRequirementAnnotation(&i.trench.ObjectMeta); na != oa {
		common.SetResourceRequirementAnnotation(&i.trench.ObjectMeta, &ret.ObjectMeta)
	}

	for x, container := range ret.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "ipam":
			if container.Image == "" {
				container.Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, imageIpam, common.Tag)
				container.ImagePullPolicy = corev1.PullAlways
			}
			if container.StartupProbe == nil {
				container.StartupProbe = common.GetProbe(common.StartUpTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			if container.ReadinessProbe == nil {
				container.ReadinessProbe = common.GetProbe(common.ReadinessTimer,
					common.GetProbeCommand(true, fmt.Sprintf(":%d", common.IpamPort), ""))
			}
			if container.LivenessProbe == nil {
				container.LivenessProbe = common.GetProbe(common.LivenessTimer,
					common.GetProbeCommand(false, "unix:///tmp/health.sock", ""))
			}
			container.Env = i.getEnvVars(ret.Spec.Template.Spec.Containers[0].Env)
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&i.trench.ObjectMeta, &container); err != nil {
				i.exec.LogInfo(fmt.Sprintf("trench %s, %v", i.trench.ObjectMeta.Name, err))
			}
		default:
			i.exec.LogError(fmt.Errorf("container %s not expected", name), "get container error")
		}
		ret.Spec.Template.Spec.Containers[x] = container
	}
	return ret
}

func (i *IpamStatefulSet) getModel() error {
	model, err := common.GetStatefulSetModel("deployment/ipam.yaml")
	if err != nil {
		return err
	}
	i.model = model
	return nil
}

func (i *IpamStatefulSet) getSelector() client.ObjectKey {
	return client.ObjectKey{
		Namespace: i.trench.ObjectMeta.Namespace,
		Name:      common.IPAMStatefulSetName(i.trench),
	}
}

func (i *IpamStatefulSet) getDesiredStatus() *appsv1.StatefulSet {
	return i.insertParameters(i.model)
}

// getReconciledDesiredStatus gets the desired status of ipam statefulset after it's created
// more paramters than what are defined in the model could be added by K8S
func (i *IpamStatefulSet) getReconciledDesiredStatus(cd *appsv1.StatefulSet) *appsv1.StatefulSet {
	template := cd.DeepCopy()
	template.Spec.Template.Spec.InitContainers = i.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Containers = i.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.Volumes = i.model.Spec.Template.Spec.Volumes
	return i.insertParameters(template)
}

func (i *IpamStatefulSet) getCurrentStatus() (*appsv1.StatefulSet, error) {
	currentStatus := &appsv1.StatefulSet{}
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

func (i *IpamStatefulSet) getAction() error {
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
