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

package common

import (
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceNamePrefixEnv = "RESOURCE_NAME_PREFIX"
	ImagePullSecretEnv    = "IMAGE_PULL_SECRET"
	NSMRegistryServiceEnv = "NSM_REGISTRY_SERVICE"
	LogLevelEnv           = "LOG_LEVEL"
	NspServiceAccountEnv  = "NSP_SERVICE_ACCOUNT"
	FeServiceAccountEnv   = "FE_SERVICE_ACCOUNT"

	Registry        = "registry.nordix.org"
	Organization    = "cloud-native/meridio"
	OrganizationNsm = "cloud-native/nsm"
	Tag             = "latest"

	BusyboxImage = "busybox"
	BusyboxTag   = "1.29"

	SubnetPoolIpv4          = "172.16.0.0/16"
	SubnetPoolIpv6          = "fd00::/48"
	ConduitPrefixLengthIpv4 = "20"
	ConduitPrefixLengthIpv6 = "56"
	NodePrefixLengthIpv4    = "24"
	NodePrefixLengthIpv6    = "64"

	NspPort        = 7778
	NspTargetPort  = 7778
	IpamPort       = 7777
	IpamTargetPort = 7777
	VlanNsePort    = 5003

	IpamSvcName             = "ipam-service"
	NspSvcName              = "nsp-service"
	PodDisruptionBudgetName = "pdb"
	LBName                  = "lb-fe"
	ProxyName               = "proxy"
	IpamName                = "ipam"
	NseName                 = "nse-vlan"
	NspName                 = "nsp"
	NspServiceAccountName   = "meridio-sa"
	CMName                  = "meridio-configuration"

	NetworkServiceName = "external-vlan"

	ResourceRequirementKey          = "resource-template"
	ResourceRequirementTemplatePath = "template/resource"
)

func NSPServiceAccountName() string {
	return os.Getenv(NspServiceAccountEnv)
}

func FEServiceAccountName() string {
	return os.Getenv(FeServiceAccountEnv)
}

func IPAMServiceName(trench *meridiov1alpha1.Trench) string {
	return GetSuffixedName(IpamSvcName, trench.ObjectMeta.Name)
}

func NSPServiceName(trench *meridiov1alpha1.Trench) string {
	return GetSuffixedName(NspSvcName, trench.ObjectMeta.Name)
}

func PDBName(attractir *meridiov1alpha1.Attractor) string {
	return GetSuffixedName(PodDisruptionBudgetName, attractir.ObjectMeta.Name)
}

func LbFeDeploymentName(attractor *meridiov1alpha1.Attractor) string {
	return GetSuffixedName(LBName, attractor.ObjectMeta.Name)
}

func ProxyDeploymentName(conduit *meridiov1alpha1.Conduit) string {
	return GetSuffixedName(ProxyName, conduit.ObjectMeta.Name)
}

func IPAMStatefulSetName(trench *meridiov1alpha1.Trench) string {
	return GetSuffixedName(IpamName, trench.ObjectMeta.Name)
}

func NSEDeploymentName(attractor *meridiov1alpha1.Attractor) string {
	return GetSuffixedName(NseName, attractor.ObjectMeta.Name)
}

func NSPStatefulSetName(trench *meridiov1alpha1.Trench) string {
	return GetSuffixedName(NspName, trench.ObjectMeta.Name)
}

func ConfigMapName(trench *meridiov1alpha1.Trench) string {
	return GetSuffixedName(CMName, trench.ObjectMeta.Name)
}

func NSPServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", NSPServiceName(cr), NspTargetPort)
}

func IPAMServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", IPAMServiceName(cr), IpamTargetPort)
}

func GetSuffixedName(resourceName, suffix string) string {
	return fmt.Sprintf("%s-%s", GetPrefixedName(resourceName), suffix)
}

func GetPrefixedName(resourceName string) string {
	return fmt.Sprintf("%s%s", getResourceNamePrefix(), resourceName)
}

func ProxyNtwkSvcNsName(cr *meridiov1alpha1.Conduit) string {
	return strings.Join([]string{ProxyName, cr.ObjectMeta.Name, cr.ObjectMeta.Labels["trench"], cr.ObjectMeta.Namespace}, ".")
}

func LoadBalancerNsName(conduit, trench, namespace string) string {
	return strings.Join([]string{conduit, trench, namespace}, ".")
}

func VlanNtwkSvcName(attractorCr *meridiov1alpha1.Attractor, trenchCr *meridiov1alpha1.Trench) string {
	return strings.Join([]string{NetworkServiceName, attractorCr.ObjectMeta.Name, trenchCr.ObjectMeta.Name, trenchCr.ObjectMeta.Namespace}, ".")
}

func getResourceNamePrefix() string {
	return os.Getenv(ResourceNamePrefixEnv)
}

func GetNSMRegistryService() string {
	return os.Getenv(NSMRegistryServiceEnv)
}

func GetLogLevel() string {
	return os.Getenv(LogLevelEnv)
}

func GetImagePullSecrets() []corev1.LocalObjectReference {
	secstr := os.Getenv(ImagePullSecretEnv)
	var pullSecs []corev1.LocalObjectReference
	if len(secstr) == 0 {
		return pullSecs
	}
	secs := strings.Split(secstr, ",")
	for _, sec := range secs {
		pullSecs = append(pullSecs, corev1.LocalObjectReference{
			Name: strings.TrimSpace(sec),
		})
	}
	return pullSecs
}

func NsName(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}

func GetExternalInterfaceName(attractor *meridiov1alpha1.Attractor) string {
	name := "ext"
	if *attractor.Spec.Interface.NSMVlan.VlanID != 0 {
		name = fmt.Sprintf("ext-vlan.%d", *attractor.Spec.Interface.NSMVlan.VlanID)
	}
	return name
}
