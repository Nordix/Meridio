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
	operatorEnv := map[string]string{
		"NSM_SERVICE_NAME": common.ProxyNtwkSvcNsName(i.conduit),
		"NSM_IPAM_SERVICE": common.IPAMServiceWithPort(i.trench),
		"NSM_NETWORK_SERVICE_NAME": common.LoadBalancerNsName(i.conduit.ObjectMeta.Name,
			i.trench.ObjectMeta.Name,
			i.conduit.ObjectMeta.Namespace),
		"NSM_IP_FAMILY":        common.GetIPFamily(i.trench),
		"NSM_TRENCH":           i.trench.GetName(),
		"NSM_CONDUIT":          i.conduit.ObjectMeta.GetName(),
		"NSM_NSP_SERVICE_NAME": common.GetPrefixedName(common.NspSvcName),
		"NSM_NSP_SERVICE_PORT": strconv.Itoa(common.NspTargetPort),
		"NSM_NAMESPACE":        i.conduit.ObjectMeta.Namespace,
		"NSM_LOG_LEVEL":        common.GetLogLevel(),
	}
	return common.CompileEnvironmentVariables(allEnv, operatorEnv)
}

func (i *Proxy) insertParameters(init *appsv1.DaemonSet) *appsv1.DaemonSet {
	// if status proxy daemonset parameters are specified in the cr, use those
	// else use the default parameters
	proxyDeploymentName := common.ProxyDeploymentName(i.conduit)
	ds := init.DeepCopy()
	ds.ObjectMeta.Name = proxyDeploymentName
	ds.ObjectMeta.Namespace = i.trench.ObjectMeta.Namespace
	ds.ObjectMeta.Labels["app"] = proxyDeploymentName
	ds.Spec.Selector.MatchLabels["app"] = proxyDeploymentName
	ds.Spec.Template.ObjectMeta.Labels["app"] = proxyDeploymentName

	imagePullSecrets := common.GetImagePullSecrets()
	if len(imagePullSecrets) > 0 {
		ds.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}

	// init container
	if ds.Spec.Template.Spec.InitContainers[0].Image == "" {
		ds.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, common.BusyboxImage, common.BusyboxTag)
	}
	ds.Spec.Template.Spec.InitContainers[0].Args = []string{
		"-c",
		common.GetProxySysCtl(i.trench),
	}

	// check resource requirement annotation update, and save annotation into deployment for visibility
	oa, _ := common.GetResourceRequirementAnnotation(&init.ObjectMeta)
	if na, _ := common.GetResourceRequirementAnnotation(&i.conduit.ObjectMeta); na != oa {
		common.SetResourceRequirementAnnotation(&i.conduit.ObjectMeta, &ds.ObjectMeta)
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
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&i.conduit.ObjectMeta, &container); err != nil {
				i.exec.LogInfo(fmt.Sprintf("conduit %s, %v", i.conduit.ObjectMeta.Name, err))
			}
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
		Name:      common.ProxyDeploymentName(i.conduit),
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
