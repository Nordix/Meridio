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
	serviceAccountName    = "meridio"
	ResourceNamePrefixEnv = "RESOURCE_NAME_PREFIX"
	ImagePullSecretEnv    = "IMAGE_PULL_SECRET"

	Registry        = "registry.nordix.org"
	Organization    = "cloud-native/meridio"
	OrganizationNsm = "cloud-native/nsm"
	Tag             = "latest"

	BusyboxImage = "busybox"
	BusyboxTag   = "1.29"

	subnetPoolIpv4   = "172.16.0.0/16"
	subnetPoolIpv6   = "fd00::/48"
	prefixLengthIpv4 = "24"
	prefixLengthIpv6 = "64"

	NspPort        = 7778
	NspTargetPort  = 7778
	ipamPort       = 7777
	ipamTargetPort = 7777

	ipamSvcName = "ipam-service"
	nspSvcName  = "nsp-service"
	lbName      = "lb-fe"
	proxyName   = "proxy"
	ipamName    = "ipam"
	nseName     = "nse-vlan"
	nspName     = "nsp"
	roleName    = "meridio-configuration-role"
	rbName      = "meridio-configuration-role-binding"
	cmName      = "meridio-configuration"

	networkServiceName = "external-vlan"
)

func ServiceAccountName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(serviceAccountName, trench)
}

func IPAMServiceName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(ipamSvcName, trench)
}

func NSPServiceName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(nspSvcName, trench)
}

func LoadBalancerDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(lbName, trench)
}

func ProxyDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(proxyName, trench)
}

func IPAMDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(ipamName, trench)
}

func NSEDeploymentName(attractor *meridiov1alpha1.Attractor) string {
	return getAttractorSuffixedName(nseName, attractor)
}

func NSPDeploymentName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(nspName, trench)
}

func RoleName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(roleName, trench)
}

func RoleBindingName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(rbName, trench)
}

func ConfigMapName(trench *meridiov1alpha1.Trench) string {
	return getTrenchSuffixedName(cmName, trench)
}

func NSPServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", NSPServiceName(cr), NspTargetPort)
}

func IPAMServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", IPAMServiceName(cr), ipamTargetPort)
}

func getTrenchSuffixedName(resourceName string, cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s%s-%s", getResourceNamePrefix(), resourceName, cr.ObjectMeta.Name)
}

func getAttractorSuffixedName(resourceName string, cr *meridiov1alpha1.Attractor) string {
	return fmt.Sprintf("%s%s-%s", getResourceNamePrefix(), resourceName, cr.ObjectMeta.Name)
}

func ProxyNtwkSvcNsName(cr *meridiov1alpha1.Trench) string {
	return strings.Join([]string{proxyName, lbName, cr.ObjectMeta.Name, cr.ObjectMeta.Namespace}, ".")
}

// alpha release: lb-fe instance is affiliated to a trench
func LoadBalancerNsName(cr *meridiov1alpha1.Trench) string {
	return strings.Join([]string{lbName, cr.ObjectMeta.Name, cr.ObjectMeta.Namespace}, ".")
}

func VlanNtwkSvcName(attr *meridiov1alpha1.Attractor) string {
	return strings.Join([]string{networkServiceName, attr.ObjectMeta.Namespace}, ".")
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
