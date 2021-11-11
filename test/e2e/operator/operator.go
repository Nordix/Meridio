/*
Copyright (c) 2021 Nordix Foundation

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

package operator

import (
	"context"
	"strings"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Operator struct {
	Namespace string
	Client    client.Client
}
type MeridioResources struct {
	Trench    meridiov1alpha1.Trench
	Attractor meridiov1alpha1.Attractor
	Conduit   meridiov1alpha1.Conduit
	Gateways  []meridiov1alpha1.Gateway
	Vips      []meridiov1alpha1.Vip
	Streams   []meridiov1alpha1.Stream
	Flows     []meridiov1alpha1.Flow
}

const (
	timeout  = time.Minute * 2
	interval = time.Second * 1
)

func (op *Operator) CleanUpResource() {
	op.CleanUpTrenches()
	op.CleanUpAttractors()
	op.CleanUpGateways()
	op.CleanUpVips()
	op.CleanUpConduits()
	op.CleanUpFlows()
	op.CleanUpStreams()
}

func (op *Operator) VerifyDeployment() {
	n := &corev1.Namespace{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{Name: op.Namespace}, n)).To(Succeed())

	dep := &appsv1.Deployment{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-controller-manager",
	}, dep)).To(Succeed())

	svc := &corev1.Service{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-webhook-service",
	}, svc)).To(Succeed())

	sa := &corev1.ServiceAccount{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-controller-manager",
	}, sa)).To(Succeed())

	lr := &rbacv1.Role{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-leader-election-role",
	}, lr)).To(Succeed())

	mr := &rbacv1.Role{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-manager-role",
	}, mr)).To(Succeed())

	lrb := &rbacv1.RoleBinding{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-leader-election-rolebinding",
	}, lrb)).To(Succeed())

	mrb := &rbacv1.RoleBinding{}
	Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: op.Namespace,
		Name:      "meridio-operator-manager-rolebinding",
	}, mrb)).To(Succeed())
}

func (op *Operator) CleanUpTrenches() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Trench{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.TrenchList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) CleanUpAttractors() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Attractor{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.AttractorList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) CleanUpVips() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Vip{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.VipList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) CleanUpGateways() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Gateway{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.GatewayList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) CleanUpConduits() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Conduit{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.ConduitList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) CleanUpStreams() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Stream{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.StreamList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) CleanUpFlows() {
	Expect(op.DeleteAllOfResources(&meridiov1alpha1.Flow{})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.FlowList{}
		err := op.ListResources(lst)
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (op *Operator) ListResources(lst client.ObjectList) error {
	return op.Client.List(context.TODO(), lst, &client.ListOptions{Namespace: op.Namespace})
}

func (op *Operator) DeleteAllOfResources(obj client.Object, opt ...client.DeleteAllOfOption) error {
	return op.Client.DeleteAllOf(context.TODO(), obj, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: op.Namespace}})
}

func (op *Operator) CreateResource(obj client.Object) error {
	return op.Client.Create(context.TODO(), obj)
}

func (op *Operator) DeleteResource(obj client.Object) error {
	return op.Client.Delete(context.TODO(), obj)
}

func (op *Operator) AssertTrenchReady(trench *meridiov1alpha1.Trench) {
	ns := trench.ObjectMeta.Namespace
	name := trench.ObjectMeta.Name
	By("checking ipam deployment")
	Eventually(func(g Gomega) {
		op.assertDeploymentReady(g, strings.Join([]string{"ipam", name}, "-"), ns)
	}, timeout, interval).Should(Succeed())

	By("checking nsp deployment")
	Eventually(func(g Gomega) {
		op.assertDeploymentReady(g, strings.Join([]string{"nsp", name}, "-"), ns)
	}, timeout, interval).Should(Succeed())

	By("checking proxy deployment")
	Eventually(func(g Gomega) {
		op.assertDaemonsetReady(g, strings.Join([]string{"proxy", name}, "-"), ns)
	}, timeout, interval).Should(Succeed())
}

func (op *Operator) AssertAttractorReady(attractor *meridiov1alpha1.Attractor) {
	ns := attractor.ObjectMeta.Namespace

	By("checking nse vlan deployment")
	Eventually(func(g Gomega) {
		op.assertDeploymentReady(g, strings.Join([]string{"nse-vlan", attractor.ObjectMeta.Name}, "-"), ns)
	}, timeout, interval).Should(Succeed())
}

func (op *Operator) AssertConduitReady(conduit *meridiov1alpha1.Conduit) {
	name := conduit.ObjectMeta.Name
	ns := conduit.ObjectMeta.Namespace

	By("checking lb-fe deployment")
	Eventually(func(g Gomega) {
		op.assertDeploymentReady(g, strings.Join([]string{"lb-fe", name}, "-"), ns)
	}, timeout, interval).Should(Succeed())
}

func (op *Operator) AssertMeridioDeploymentsReady(m MeridioResources) {
	op.AssertTrenchReady(&m.Trench)
	op.AssertAttractorReady(&m.Attractor)
	op.AssertConduitReady(&m.Conduit)
}

func (op *Operator) assertDeploymentReady(g Gomega, name, ns string) {
	dep := &appsv1.Deployment{}
	// checking if the deployment exists
	g.Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, dep)).Should(Succeed())
	g.Expect(dep).ToNot(BeNil())

	// checking all replicas are ready
	g.Expect(dep.Status.ReadyReplicas).To(Equal(dep.Status.Replicas))

	// checking all pods are ready and never restarted
	listOptions := &client.ListOptions{
		LabelSelector: labels.Set(dep.Labels).AsSelector(),
	}
	op.podsRunning(g, listOptions)
}

func (op *Operator) podsRunning(g Gomega, opts client.ListOption) bool {
	pods := &corev1.PodList{}
	for _, pod := range pods.Items {
		g.Expect(op.Client.List(context.Background(), pods, opts)).Should(Succeed())
		// wait for all the pods of the deployment are in running status
		g.Expect(pod.Status.Phase).Should(Equal(corev1.PodRunning))
		// check the restart count of each container of each pod
		for _, container := range pod.Status.ContainerStatuses {
			g.Expect(container.RestartCount).To(Equal(int32(0)))
		}
	}
	return true
}

func (op *Operator) assertDaemonsetReady(g Gomega, name, ns string) {
	ds := &appsv1.DaemonSet{}
	// checking if the daemonset exists
	g.Expect(op.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, ds)).Should(Succeed())

	// checking all desired replicas are ready"
	g.Expect(ds.Status.NumberReady).To(Equal(ds.Status.DesiredNumberScheduled))
	listOptions := &client.ListOptions{
		LabelSelector: labels.Set(ds.Labels).AsSelector(),
	}
	op.podsRunning(g, listOptions)
}

func (op *Operator) DeployMeridio(m MeridioResources) {
	// create meridio instances by creating custom resources
	By("creating custom resources")
	Expect(op.CreateResource(&m.Trench)).To(Succeed())
	op.AssertTrenchReady(&m.Trench)

	Expect(op.CreateResource(&m.Attractor)).To(Succeed())
	op.AssertAttractorReady(&m.Attractor)

	Expect(op.CreateResource(&m.Conduit)).To(Succeed())
	op.AssertConduitReady(&m.Conduit)

	for _, g := range m.Gateways {
		Expect(op.CreateResource(&g)).To(Succeed())
	}
	for _, v := range m.Vips {
		Expect(op.CreateResource(&v)).To(Succeed())
	}
	for _, s := range m.Streams {
		Expect(op.CreateResource(&s)).To(Succeed())
	}
	for _, f := range m.Flows {
		Expect(op.CreateResource(&f)).To(Succeed())
	}
}

func (op Operator) UndeployMerido(m MeridioResources) {
	By("deleting custom resources")
	Expect(op.DeleteResource(&m.Trench)).To(Succeed())
	Expect(op.DeleteResource(&m.Attractor)).To(Succeed())
	Expect(op.DeleteResource(&m.Conduit)).To(Succeed())

	for _, g := range m.Gateways {
		Expect(op.DeleteResource(&g)).To(Succeed())
	}
	for _, v := range m.Vips {
		Expect(op.DeleteResource(&v)).To(Succeed())
	}
	for _, s := range m.Streams {
		Expect(op.DeleteResource(&s)).To(Succeed())
	}
	for _, f := range m.Flows {
		Expect(op.DeleteResource(&f)).To(Succeed())
	}
}
