/*
Copyright (c) 2025 OpenInfra Foundation Europe

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
	"context"
	"testing"

	"github.com/go-logr/logr/testr"
	meridiov1 "github.com/nordix/meridio/api/v1"
	"github.com/nordix/meridio/pkg/controllers/common"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Note: The fake client simulates a Kubernetes API server using an in-memory
// object store, thus it does not replicate all API server functionalities.

func newTestTrench() *meridiov1.Trench {
	return &meridiov1.Trench{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-trench",
			Namespace: "default",
		},
		Spec: meridiov1.TrenchSpec{
			IPFamily: string(meridiov1.Dualstack),
		},
	}
}

func newTestAttractor() *meridiov1.Attractor {
	return &meridiov1.Attractor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-attractor",
			Namespace: "default",
			Labels:    map[string]string{"trench": "test-trench"},
		},
		Spec: meridiov1.AttractorSpec{
			Replicas:   &[]int32{1}[0],
			Composites: []string{"test-conduit"},
			Interface: meridiov1.InterfaceSpec{
				Name: "ext-vlan",
				Type: meridiov1.NSMVlan,
				NSMVlan: meridiov1.NSMVlanSpec{
					BaseInterface: "eth0",
					VlanID:        &[]int32{100}[0],
				},
			},
		},
	}
}

func newPodAntiAffinity(name string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{name},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}
}

// newLbFeModel returns a simplified stateless-lb-frontend model, i.e. what the
// operator loads from the deployment template.
func newLbFeModel(securityContext *corev1.PodSecurityContext) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "stateless-lb-frontend",
			Labels: map[string]string{"app": "stateless-lb-frontend"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "stateless-lb-frontend"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "stateless-lb-frontend"},
				},
				Spec: corev1.PodSpec{
					Affinity: newPodAntiAffinity("stateless-lb-frontend"),
					InitContainers: []corev1.Container{
						{
							Name:    "sysctl-init",
							Command: []string{"/bin/sh"},
						},
					},
					Containers: []corev1.Container{
						{Name: "stateless-lb"},
						{Name: "nsc"},
						{Name: "frontend"},
					},
					SecurityContext: securityContext,
				},
			},
		},
	}
}

// newLbFeDeployment returns a stateless-lb-frontend deployment as it could have
// been created by an earlier version of the operator, i.e. lacking the POD level
// security context.
func newLbFeDeployment(attr *meridiov1.Attractor) *appsv1.Deployment {
	name := common.LbFeDeploymentName(attr)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: attr.ObjectMeta.Namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: attr.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Affinity: newPodAntiAffinity(name),
					InitContainers: []corev1.Container{
						{
							Name:    "sysctl-init",
							Command: []string{"/bin/sh"},
						},
					},
					Containers: []corev1.Container{
						{Name: "stateless-lb"},
						{Name: "nsc"},
						{Name: "frontend"},
					},
				},
			},
		},
	}
}

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = meridiov1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func newTestLoadBalancer(t *testing.T, c client.Client, model *appsv1.Deployment, attr *meridiov1.Attractor, trench *meridiov1.Trench) *LoadBalancer {
	scheme := newTestScheme()
	logger := testr.New(t)
	lbFeDeploymentName = common.LbFeDeploymentName(attr)
	return &LoadBalancer{
		model:     model,
		trench:    trench,
		attractor: attr,
		exec:      common.NewExecutor(scheme, c, context.Background(), trench, logger.WithName("test")),
	}
}

func getLbFeDeployment(t *testing.T, c client.Client, attr *meridiov1.Attractor) *appsv1.Deployment {
	dep := &appsv1.Deployment{}
	err := c.Get(context.Background(),
		types.NamespacedName{Name: common.LbFeDeploymentName(attr), Namespace: attr.ObjectMeta.Namespace},
		dep)
	assert.NoError(t, err)
	return dep
}

// TestLoadBalancer_PodSecurityContext verifies that the POD level security
// context (sysctls, fsGroup, ...) of the model is applied when updating an
// already existing stateless-lb-frontend deployment lacking it.
func TestLoadBalancer_PodSecurityContext(t *testing.T) {
	trench := newTestTrench()
	attr := newTestAttractor()
	sysctls := []corev1.Sysctl{
		{Name: "net.ipv4.conf.all.forwarding", Value: "1"},
		{Name: "net.ipv6.conf.all.forwarding", Value: "1"},
	}
	model := newLbFeModel(&corev1.PodSecurityContext{
		FSGroup: &[]int64{2000}[0],
		Sysctls: sysctls,
	})

	old := newLbFeDeployment(attr)
	assert.Nil(t, old.Spec.Template.Spec.SecurityContext)
	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme()).WithObjects(trench, attr, old).Build()

	cs := getLbFeDeployment(t, fakeClient, attr)
	lb := newTestLoadBalancer(t, fakeClient, model, attr, trench)
	assert.NoError(t, lb.getAction())
	assert.NoError(t, lb.exec.RunActions())

	dep := getLbFeDeployment(t, fakeClient, attr)
	assert.False(t, equality.Semantic.DeepEqual(dep.Spec, cs.Spec)) // update executed
	if assert.NotNil(t, dep.Spec.Template.Spec.SecurityContext) {
		assert.Equal(t, sysctls, dep.Spec.Template.Spec.SecurityContext.Sysctls)
		assert.Equal(t, int64(2000), *dep.Spec.Template.Spec.SecurityContext.FSGroup)
	}

	// Reconciling again must not trigger a new update (no endless update loop)
	cs = getLbFeDeployment(t, fakeClient, attr)
	lb = newTestLoadBalancer(t, fakeClient, model, attr, trench)
	assert.NoError(t, lb.getAction())
	assert.NoError(t, lb.exec.RunActions())

	dep = getLbFeDeployment(t, fakeClient, attr)
	assert.True(t, equality.Semantic.DeepEqual(dep.Spec, cs.Spec)) // update not executed
}

// TestLoadBalancer_NoPodSecurityContext verifies that no update is triggered for
// an existing stateless-lb-frontend deployment if the model has no POD level
// security context either (nil and empty are semantically the same).
func TestLoadBalancer_NoPodSecurityContext(t *testing.T) {
	trench := newTestTrench()
	attr := newTestAttractor()
	model := newLbFeModel(nil)

	old := newLbFeDeployment(attr)
	fakeClient := fake.NewClientBuilder().WithScheme(newTestScheme()).WithObjects(trench, attr, old).Build()

	// first update aligns the deployment with the model
	lb := newTestLoadBalancer(t, fakeClient, model, attr, trench)
	assert.NoError(t, lb.getAction())
	assert.NoError(t, lb.exec.RunActions())
	dep := getLbFeDeployment(t, fakeClient, attr)
	assert.Nil(t, dep.Spec.Template.Spec.SecurityContext)

	// no further update expected
	cs := getLbFeDeployment(t, fakeClient, attr)
	lb = newTestLoadBalancer(t, fakeClient, model, attr, trench)
	assert.NoError(t, lb.getAction())
	assert.NoError(t, lb.exec.RunActions())
	dep = getLbFeDeployment(t, fakeClient, attr)
	assert.True(t, equality.Semantic.DeepEqual(dep.Spec, cs.Spec)) // update not executed
	assert.Nil(t, dep.Spec.Template.Spec.SecurityContext)
}
