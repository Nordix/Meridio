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
	"encoding/json"
	"fmt"

	meridiov1 "github.com/nordix/meridio/api/v1"
	common "github.com/nordix/meridio/pkg/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lbImage = "stateless-lb"
	feImage = "frontend"
)

var lbFeDeploymentName string

type LoadBalancer struct {
	model     *appsv1.Deployment
	trench    *meridiov1.Trench
	attractor *meridiov1.Attractor
	exec      *common.Executor
}

func (l *LoadBalancer) getModel() error {
	model, err := common.GetDeploymentModel("deployment/stateless-lb-frontend.yaml")
	if err != nil {
		return err
	}
	l.model = model
	return nil
}

func NewLoadBalancer(e *common.Executor, attr *meridiov1.Attractor, t *meridiov1.Trench) (*LoadBalancer, error) {
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
	operatorEnv := []corev1.EnvVar{
		{Name: "NSM_SERVICE_NAME", Value: common.LoadBalancerNsName(l.attractor.Spec.Composites[0],
			l.trench.ObjectMeta.Name,
			l.attractor.ObjectMeta.Namespace)},
		{Name: "NSM_CONDUIT_NAME", Value: l.attractor.Spec.Composites[0]},
		{Name: "NSM_TRENCH_NAME", Value: l.trench.ObjectMeta.Name},
		{Name: "NSM_NSP_SERVICE", Value: common.NSPServiceWithPort(l.trench)},
		{Name: "NSM_LOG_LEVEL", Value: common.GetLogLevel()},
	}
	if rpcTimeout := common.GetGRPCProbeRPCTimeout(); rpcTimeout != "" {
		operatorEnv = append(operatorEnv, corev1.EnvVar{Name: "NSM_GRPC_PROBE_RPC_TIMEOUT", Value: rpcTimeout})
	}
	return common.CompileEnvironmentVariables(allEnv, operatorEnv)
}

func (l *LoadBalancer) getNscEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	operatorEnv := []corev1.EnvVar{
		{Name: "NSM_NETWORK_SERVICES", Value: fmt.Sprintf("kernel://%s/%s", common.VlanNtwkSvcName(l.attractor, l.trench), l.attractor.Spec.Interface.Name)},
		{Name: "NSM_LOG_LEVEL", Value: common.GetLogLevel()},
		{Name: "NSM_LIVENESSCHECKENABLED", Value: "false"},
	}
	return common.CompileEnvironmentVariables(allEnv, operatorEnv)
}

func (l *LoadBalancer) getFeEnvVars(allEnv []corev1.EnvVar) []corev1.EnvVar {
	operatorEnv := []corev1.EnvVar{
		{Name: "NFE_CONFIG_MAP_NAME", Value: common.ConfigMapName(l.trench)},
		{Name: "NFE_NSP_SERVICE", Value: common.NSPServiceWithPort(l.trench)},
		{Name: "NFE_TRENCH_NAME", Value: l.trench.ObjectMeta.Name},
		{Name: "NFE_ATTRACTOR_NAME", Value: l.attractor.ObjectMeta.Name},
		{Name: "NFE_NAMESPACE", Value: l.attractor.ObjectMeta.Namespace},
		{Name: "NFE_EXTERNAL_INTERFACE", Value: func() string {
			externalInterface := l.attractor.Spec.Interface.Name
			// if set use the interface provided by the Network Attachment
			if l.attractor.Spec.Interface.Type == meridiov1.NAD &&
				len(l.attractor.Spec.Interface.NetworkAttachments) == 1 &&
				l.attractor.Spec.Interface.NetworkAttachments[0].InterfaceRequest != "" {
				externalInterface = l.attractor.Spec.Interface.NetworkAttachments[0].InterfaceRequest
			}
			return externalInterface
		}()},
		{Name: "NFE_LOG_LEVEL", Value: common.GetLogLevel()},
	}
	return common.CompileEnvironmentVariables(allEnv, operatorEnv)
}

// Appends network annotation(s) based on the Network Attachment configuration in Attractor.
// Currently a single Network Attachment is supported, but the code can be easily extended
// to support multiple.
func (l *LoadBalancer) insertNetworkAnnotation(dep *appsv1.Deployment) error {
	if len(l.attractor.Spec.Interface.NetworkAttachments) != 1 {
		return fmt.Errorf("required one network attachment")
	}

	if dep.Spec.Template.ObjectMeta.Annotations == nil {
		dep.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}

	// parse existing annotations
	netAttachSels, err := common.GetNetworkAnnotation(
		dep.Spec.Template.ObjectMeta.Annotations[common.NetworkAttachmentAnnot],
		l.attractor.ObjectMeta.Namespace,
	)
	if err != nil {
		return err
	}

	netAttachSelMap := common.MakeNetworkAttachmentSpecMap(netAttachSels) // convert to map to check for duplicates
	// check if attractor defined network attachments are already present among existing annotations
	attrNetAttachSels := []*common.NetworkAttachmentSelector{}
	for _, na := range l.attractor.Spec.Interface.NetworkAttachments {
		if na.Namespace == "" {
			return fmt.Errorf("namespace not specified")
		}
		sel := common.NetworkAttachmentSelector{
			Name:      na.Name,
			Namespace: na.Namespace,
			InterfaceRequest: func() string {
				if na.InterfaceRequest == "" {
					return l.attractor.Spec.Interface.Name // if missing use the interface name provided by attractor.Spec.Interface
				} else {
					return na.InterfaceRequest
				}
			}(),
		}
		if _, ok := netAttachSelMap[sel]; ok {
			continue // already present among existing annotations
		}
		attrNetAttachSels = append(attrNetAttachSels, &sel)
	}

	if len(attrNetAttachSels) != 0 {
		// append attractor defined network attachments to existing ones using json encoding
		netAttachSels = append(netAttachSels, attrNetAttachSels...)
		enc, err := json.Marshal(netAttachSels)
		if err != nil {
			return err
		}
		dep.Spec.Template.ObjectMeta.Annotations[common.NetworkAttachmentAnnot] = string(enc)
	}

	return nil
}

func (l *LoadBalancer) insertParameters(dep *appsv1.Deployment) *appsv1.Deployment {
	// if status stateless-lb-frontend deployment parameters are specified in the cr, use those
	// else use the default parameters
	ret := dep.DeepCopy()
	ret.ObjectMeta.Name = lbFeDeploymentName
	ret.ObjectMeta.Namespace = l.attractor.ObjectMeta.Namespace
	ret.ObjectMeta.Labels["app"] = lbFeDeploymentName
	ret.Spec.Selector.MatchLabels["app"] = lbFeDeploymentName

	ret.Spec.Replicas = l.attractor.Spec.Replicas

	ret.Spec.Template.ObjectMeta.Labels["app"] = lbFeDeploymentName
	ret.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0] = lbFeDeploymentName
	ret.Spec.Template.Spec.ServiceAccountName = common.FEServiceAccountName()
	imagePullSecrets := common.GetImagePullSecrets()
	if len(imagePullSecrets) > 0 {
		ret.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}

	if l.attractor.Spec.Interface.Type == meridiov1.NAD {
		if err := l.insertNetworkAnnotation(ret); err != nil {
			l.exec.LogError(err, fmt.Sprintf("attractor %s, network attachment annotation failure", l.attractor.ObjectMeta.Name))
		}
	}

	if ret.Spec.Template.Spec.InitContainers[0].Image == "" {
		ret.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s/%s:%s", common.Registry, common.Organization, common.BusyboxImage, common.BusyboxTag)
		ret.Spec.Template.Spec.InitContainers[0].ImagePullPolicy = corev1.PullIfNotPresent
	}
	ret.Spec.Template.Spec.InitContainers[0].Args = []string{
		"-c",
		common.GetLoadBalancerSysCtl(l.trench),
	}

	// check resource requirement annotation update, and save annotation into deployment for visibility
	oa, _ := common.GetResourceRequirementAnnotation(&dep.ObjectMeta)
	if na, _ := common.GetResourceRequirementAnnotation(&l.attractor.ObjectMeta); na != oa {
		common.SetResourceRequirementAnnotation(&l.attractor.ObjectMeta, &ret.ObjectMeta)
	}

	// nsc container not needed if interface type is not nsm-vlan
	if l.attractor.Spec.Interface.Type != meridiov1.NSMVlan {
		for i, container := range ret.Spec.Template.Spec.Containers {
			if container.Name == "nsc" {
				clen := len(ret.Spec.Template.Spec.Containers)
				ret.Spec.Template.Spec.Containers[i] = ret.Spec.Template.Spec.Containers[clen-1]
				ret.Spec.Template.Spec.Containers = ret.Spec.Template.Spec.Containers[:clen-1]
				break
			}

		}
	}

	for i, container := range ret.Spec.Template.Spec.Containers {
		switch name := container.Name; name {
		case "stateless-lb":
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
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&l.attractor.ObjectMeta, &container); err != nil {
				l.exec.LogInfo(fmt.Sprintf("attractor %s, %v", l.attractor.ObjectMeta.Name, err))
			}
		case "nsc":
			if container.Image == "" {
				container.Image = "registry.nordix.org/cloud-native/nsm/cmd-nsc:v1.11.0"
				container.ImagePullPolicy = corev1.PullAlways
			}
			container.Env = l.getNscEnvVars(container.Env)
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&l.attractor.ObjectMeta, &container); err != nil {
				l.exec.LogInfo(fmt.Sprintf("attractor %s, %v", l.attractor.ObjectMeta.Name, err))
			}
		case "frontend":
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
			// set resource requirements for container (if not found, then values from model
			// are kept even upon updates, as getReconciledDesiredStatus() overwrites containers)
			if err := common.SetContainerResourceRequirements(&l.attractor.ObjectMeta, &container); err != nil {
				l.exec.LogInfo(fmt.Sprintf("attractor %s, %v", l.attractor.ObjectMeta.Name, err))
			}
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

// getReconciledDesiredStatus gets the desired status of stateless-lb-frontend deployment after it's created
// more paramters than what are defined in the model could be added by K8S
func (l *LoadBalancer) getReconciledDesiredStatus(lb *appsv1.Deployment) *appsv1.Deployment {
	template := lb.DeepCopy()
	template.Spec.Template.Spec.InitContainers = l.model.Spec.Template.Spec.InitContainers
	template.Spec.Template.Spec.Containers = l.model.Spec.Template.Spec.Containers
	template.Spec.Template.Spec.Volumes = l.model.Spec.Template.Spec.Volumes
	return l.insertParameters(template)
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
