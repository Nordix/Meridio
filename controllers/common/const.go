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
	SAName                = "meridio-sa"
	ResourceNamePrefixEnv = "RESOURCE_NAME_PREFIX"
	ImagePullSecretEnv    = "IMAGE_PULL_SECRET"

	Registry        = "registry.nordix.org"
	Organization    = "cloud-native/meridio"
	OrganizationNsm = "cloud-native/nsm"
	Tag             = "latest"

	BusyboxImage = "busybox"
	BusyboxTag   = "1.29"

	SubnetPoolIpv4   = "172.16.0.0/16"
	SubnetPoolIpv6   = "fd00::/48"
	PrefixLengthIpv4 = "24"
	PrefixLengthIpv6 = "64"

	NspPort        = 7778
	NspTargetPort  = 7778
	IpamPort       = 7777
	IpamTargetPort = 7777

	IpamSvcName = "ipam-service"
	NspSvcName  = "nsp-service"
	LBName      = "lb-fe"
	ProxyName   = "proxy"
	IpamName    = "ipam"
	NseName     = "nse-vlan"
	NspName     = "nsp"
	RlName      = "meridio-configuration-role"
	RBName      = "meridio-configuration-role-binding"
	CMName      = "meridio-configuration"

	NetworkServiceName = "external-vlan"
)

func ServiceAccountName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(SAName, trench)
}

func IPAMServiceName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(IpamSvcName, trench)
}

func NSPServiceName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(NspSvcName, trench)
}

func LoadBalancerDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(LBName, trench)
}

func ProxyDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(ProxyName, trench)
}

func IPAMDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(IpamName, trench)
}

func NSEDeploymentName(attractor *meridiov1alpha1.Attractor) string {
	return getAttractorSuffixedName(NseName, attractor)
}

func NSPDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(NspName, trench)
}

func RoleName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(RlName, trench)
}

func RoleBindingName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(RBName, trench)
}

func ConfigMapName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(CMName, trench)
}

func NSPServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", NSPServiceName(cr), NspTargetPort)
}

func IPAMServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", IPAMServiceName(cr), IpamTargetPort)
}

func getTrenchSuffixedName(resourceName string, cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s%s-%s", getResourceNamePrefix(), resourceName, cr.ObjectMeta.Name)
}

func getAttractorSuffixedName(resourceName string, cr *meridiov1alpha1.Attractor) string {
	return fmt.Sprintf("%s%s-%s", getResourceNamePrefix(), resourceName, cr.ObjectMeta.Name)
}

func ProxyNtwkSvcNsName(cr *meridiov1alpha1.Trench) string {
	return strings.Join([]string{ProxyName, LBName, cr.ObjectMeta.Name, cr.ObjectMeta.Namespace}, ".")
}

// alpha release: lb-fe instance is affiliated to a trench
func LoadBalancerNsName(cr *meridiov1alpha1.Trench) string {
	return strings.Join([]string{LBName, cr.ObjectMeta.Name, cr.ObjectMeta.Namespace}, ".")
}

func VlanNtwkSvcName(attr *meridiov1alpha1.Attractor) string {
	return strings.Join([]string{NetworkServiceName, attr.ObjectMeta.Namespace}, ".")
}

func getResourceNamePrefix() string {
	return os.Getenv(ResourceNamePrefixEnv)
}

func GetImagePullSecrets() []corev1.LocalObjectReference {
	secstr := os.Getenv(ImagePullSecretEnv)
	secs := strings.Split(secstr, ",")
	var pullSecs []corev1.LocalObjectReference
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
