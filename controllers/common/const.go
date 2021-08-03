package common

import (
	"fmt"
	"os"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceAccountName    = "meridio"
	ResourceNamePrefixEnv = "RESOURCE_NAME_PREFIX"

	Registry        = "registry.nordix.org"
	Organization    = "cloud-native/meridio"
	OrganizationNsm = "cloud-native/nsm"
	Tag             = "latest"
	PullPolicy      = "IfNotPresent"

	IPv4      ipFamily = "ipv4"
	IPv6      ipFamily = "ipv6"
	Dualstack ipFamily = "dualstack"

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

	proxyNetworkService = "proxy.lb-fe"
)

func ServiceAccountName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, serviceAccountName)
}

func IPAMServiceName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, ipamSvcName)
}

func NSPServiceName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, nspSvcName)
}

func LoadBalancerDeploymentName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, lbName)
}

func ProxyDeploymentName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, proxyName)
}

func IPAMDeploymentName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, ipamName)
}

func NSEDeploymentName(cr *meridiov1alpha1.Attractor) string {
	return getFullName(&cr.ObjectMeta, nseName)
}

func NSPDeploymentName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, nspName)
}

func RoleName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, roleName)
}

func RoleBindingName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, rbName)
}

func ConfigMapName(cr *meridiov1alpha1.Trench) string {
	return getFullName(&cr.ObjectMeta, cmName)
}

func NSPServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", NSPServiceName(cr), NspTargetPort)
}

func IPAMServiceWithPort(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s:%d", IPAMServiceName(cr), ipamTargetPort)
}

func ProxyNtwkSvcNsName(cr *meridiov1alpha1.Trench) string {
	return getAppNsName(proxyNetworkService, cr.ObjectMeta)
}

// alpha release: lb-fe instance is affiliated to a trench
func LoadBalancerNsName(cr *meridiov1alpha1.Trench) string {
	return getAppNsName(LoadBalancerDeploymentName(cr), cr.ObjectMeta)
}

func NSENsName(attr *meridiov1alpha1.Attractor) string {
	return getAppNsName(NSEDeploymentName(attr), attr.ObjectMeta)
}

func getFullName(meta *metav1.ObjectMeta, resourceName string) string {
	return fmt.Sprintf("%s%s-%s", getResourceNamePrefix(), resourceName, meta.Name)
}

func getAppNsName(app string, meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%s.%s.%s", app, meta.Name, meta.Namespace)
}

func getResourceNamePrefix() string {
	return os.Getenv(ResourceNamePrefixEnv)
}

func NsName(meta metav1.ObjectMeta) string {
	return fmt.Sprintf("%s/%s", meta.Namespace, meta.Name)
}
