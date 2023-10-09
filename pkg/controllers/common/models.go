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

	meridiov1 "github.com/nordix/meridio/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func GetIPFamily(cr *meridiov1.Trench) string {
	return string(cr.Spec.IPFamily)
}

const ipv4SysCtl = "sysctl -w net.ipv4.conf.all.forwarding=1 ; sysctl -w net.ipv4.fib_multipath_hash_policy=1 ; sysctl -w net.ipv4.conf.all.rp_filter=0 ; sysctl -w net.ipv4.conf.default.rp_filter=0"
const ipv6SysCtl = "sysctl -w net.ipv6.conf.all.forwarding=1 ; sysctl -w net.ipv6.fib_multipath_hash_policy=1"
const localPortRangeSysCtl = "sysctl -w net.ipv4.ip_local_port_range='49152 65535'" // ephemeral port range to make BIRD use IANA approved src ports for BFD
const ipv4FwmarkReflect = "sysctl -w net.ipv4.fwmark_reflect=1"
const ipv6FwmarkReflect = "sysctl -w net.ipv6.fwmark_reflect=1"

func GetLoadBalancerSysCtl(cr *meridiov1.Trench) string {
	if cr.Spec.IPFamily == string(meridiov1.Dualstack) {
		return fmt.Sprintf("%s ; %s ; %s ; %s ; %s", ipv4SysCtl, ipv6SysCtl, localPortRangeSysCtl, ipv4FwmarkReflect, ipv6FwmarkReflect)
	} else if cr.Spec.IPFamily == string(meridiov1.IPv4) {
		return fmt.Sprintf("%s ; %s; %s", ipv4SysCtl, localPortRangeSysCtl, ipv4FwmarkReflect)
	} else if cr.Spec.IPFamily == string(meridiov1.IPv6) {
		return fmt.Sprintf("%s ; %s ; %s", ipv6SysCtl, localPortRangeSysCtl, ipv6FwmarkReflect)
	}
	return ""
}

func GetProxySysCtl(cr *meridiov1.Trench) string {
	if cr.Spec.IPFamily == string(meridiov1.Dualstack) {
		return fmt.Sprintf("%s ; %s ; %s", ipv4SysCtl, ipv6SysCtl, "sysctl -w net.ipv6.conf.all.accept_dad=0")
	} else if cr.Spec.IPFamily == string(meridiov1.IPv6) {
		return fmt.Sprintf("%s ; %s", ipv6SysCtl, "sysctl -w net.ipv6.conf.all.accept_dad=0")
	} else if cr.Spec.IPFamily == string(meridiov1.IPv4) {
		return ipv4SysCtl
	}
	return ""
}

type probeTimer struct {
	initialDelaySeconds int32
	periodSeconds       int32
	timeoutSeconds      int32
	failureThreshold    int32
	successThreshold    int32
}

var (
	LivenessTimer = probeTimer{
		initialDelaySeconds: 0,
		periodSeconds:       10,
		timeoutSeconds:      3,
		failureThreshold:    5,
		successThreshold:    1,
	}

	ReadinessTimer = probeTimer{
		initialDelaySeconds: 0,
		periodSeconds:       10,
		timeoutSeconds:      3,
		failureThreshold:    5,
		successThreshold:    1,
	}

	StartUpTimer = probeTimer{
		initialDelaySeconds: 0,
		periodSeconds:       2,
		timeoutSeconds:      2,
		failureThreshold:    30,
		successThreshold:    1,
	}
)

func GetProbeCommand(spiffe bool, addr, svc string) []string {
	ret := []string{
		"/bin/grpc_health_probe",
		fmt.Sprintf("-addr=%s", addr),
		fmt.Sprintf("-service=%s", svc),
		"-connect-timeout=100ms",
		"-rpc-timeout=150ms",
	}
	if spiffe {
		ret = append(ret, "-spiffe")
	}
	return ret
}

func GetProbe(timer probeTimer, command []string) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: command,
			},
		},
		InitialDelaySeconds: timer.initialDelaySeconds,
		PeriodSeconds:       timer.periodSeconds,
		TimeoutSeconds:      timer.timeoutSeconds,
		FailureThreshold:    timer.failureThreshold,
		SuccessThreshold:    timer.successThreshold,
	}
}

func GetDeploymentModel(f string) (*appsv1.Deployment, error) {
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

func GetStatefulSetModel(f string) (*appsv1.StatefulSet, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	deployment := &appsv1.StatefulSet{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(deployment)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return deployment, nil
}

func GetDaemonsetModel(f string) (*appsv1.DaemonSet, error) {
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

func GetServiceModel(f string) (*corev1.Service, error) {
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

func GetRoleModel(f string) (*rbacv1.Role, error) {
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

func GetRoleBindingModel(f string) (*rbacv1.RoleBinding, error) {
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

func GetServiceAccountModel(f string) (*corev1.ServiceAccount, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	rb := &corev1.ServiceAccount{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(rb)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return rb, nil
}

func GetTrenchBySelector(e *Executor, selector client.ObjectKey) (*meridiov1.Trench, error) {
	trench := &meridiov1.Trench{}
	err := e.GetObject(selector, trench)
	return trench, err
}

func GetPodDisruptionBudgetModel(f string) (*policyv1.PodDisruptionBudget, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	rb := &policyv1.PodDisruptionBudget{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(rb)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return rb, nil
}

func CompileEnvironmentVariables(allEnv []corev1.EnvVar, operatorEnv []corev1.EnvVar) []corev1.EnvVar {
	operatorEnvMap := map[string]string{}
	for _, env := range operatorEnv {
		operatorEnvMap[env.Name] = env.Value
	}
	res := []corev1.EnvVar{}
	for _, env := range allEnv {
		value, isOperatorEnv := operatorEnvMap[env.Name]
		if isOperatorEnv {
			if env.Value == "" { // set only if no value already set
				env.Value = value
			}
			delete(operatorEnvMap, env.Name)
			res = append(res, env)
			continue
		}
		res = append(res, env)
	}
	// Add missing operator env variables
	for _, env := range operatorEnv {
		_, isOperatorEnv := operatorEnvMap[env.Name]
		if !isOperatorEnv {
			continue
		}
		res = append(res, env)
	}
	return res
}
