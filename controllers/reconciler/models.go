package reconciler

import (
	"fmt"
	"os"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	Registry        = "registry.nordix.org"
	Organization    = "cloud-native/meridio"
	OrganizationNsm = "cloud-native/nsm"
	Tag             = "latest"
	PullPolicy      = "IfNotPresent"
	busyboxImage    = "busybox"
	busyboxTag      = "1.29"

	IPv4      ipFamily = "ipv4"
	IPv6      ipFamily = "ipv6"
	Dualstack ipFamily = "dualstack"

	vipIpv4          = "20.0.0.1/32"
	vipIpv6          = "2000::1/128"
	subnetPoolIpv4   = "172.16.0.0/16"
	subnetPoolIpv6   = "fd00::/48"
	prefixLengthIpv4 = "24"
	prefixLengthIpv6 = "64"

	proxyNetworkService = "proxy.load-balancer"
	lbNetworkService    = "load-balancer"
	vlanNetworkService  = "external-vlan"
	vlanItf             = "eth0"
	vlanID              = "100"
	vlanPrefixV4        = "169.254.100.0/24"
	vlanPrefixV6        = "100:100::/64"
)

type ipFamily string

func getVips(cr *meridiov1alpha1.Trench) string {
	ipFamily := IPv4
	if ipFamily == IPv4 {
		return vipIpv4
	} else if ipFamily == IPv6 {
		return vipIpv6
	} else if ipFamily == Dualstack {
		return fmt.Sprintf("%s,%s", vipIpv4, vipIpv6)
	}
	return ""
}

func getSubnetPool(cr *meridiov1alpha1.Trench) string {
	ipFamily := IPv4
	if ipFamily == IPv4 {
		return subnetPoolIpv4
	} else if ipFamily == IPv6 {
		return subnetPoolIpv6
	} else if ipFamily == Dualstack {
		return fmt.Sprintf("%s,%s", subnetPoolIpv4, subnetPoolIpv6)
	}
	return ""
}

func getPrefixLength(cr *meridiov1alpha1.Trench) string {
	ipFamily := IPv4
	if ipFamily == IPv4 {
		return prefixLengthIpv4
	} else if ipFamily == IPv6 {
		return prefixLengthIpv6
	} else if ipFamily == Dualstack {
		return fmt.Sprintf("%s,%s", prefixLengthIpv4, prefixLengthIpv4)
	}
	return ""
}

func getVlanNsName(cr *meridiov1alpha1.Trench) string {
	return fmt.Sprintf("%s.%s", vlanNetworkService, cr.ObjectMeta.Namespace)
}

func GetReadinessProbe(cr *meridiov1alpha1.Trench) *corev1.Probe {
	// if readiness probe is set in the cr do something
	// else use the default readiness probe
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/grpc_health_probe", "-addr=:8000", "-connect-timeout=100ms", "-rpc-timeout=150ms"},
			},
		},
		InitialDelaySeconds: 0,
		PeriodSeconds:       10,
		TimeoutSeconds:      3,
		FailureThreshold:    5,
	}
}

func GetLivenessProbe(cr *meridiov1alpha1.Trench) *corev1.Probe {
	// if liveness probe is set in the cr do something
	// else use the default liveness probe
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/grpc_health_probe", "-addr=:8000", "-connect-timeout=100ms", "-rpc-timeout=150ms"},
			},
		},
		InitialDelaySeconds: 0,
		PeriodSeconds:       10,
		TimeoutSeconds:      3,
		FailureThreshold:    5,
	}
}

func getDeploymentModel(f string) (*appsv1.Deployment, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	deployment := &appsv1.Deployment{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(deployment)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return deployment, nil
}

func getDaemonsetModel(f string) (*appsv1.DaemonSet, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	ds := &appsv1.DaemonSet{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(ds)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return ds, nil
}

func getServiceModel(f string) (*corev1.Service, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	service := &corev1.Service{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(service)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return service, nil
}

func getRoleModel(f string) (*rbacv1.Role, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	role := &rbacv1.Role{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(role)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return role, nil
}

func getRoleBindingModel(f string) (*rbacv1.RoleBinding, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	rb := &rbacv1.RoleBinding{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(rb)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return rb, nil
}
