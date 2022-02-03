package e2e

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	scalescheme "k8s.io/client-go/scale/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	timeout  = time.Minute * 2
	interval = time.Second * 1

	trenchName    = "trench-a"
	attractorName = "attr-1"

	TrenchCRDName    = "trenches.meridio.nordix.org"
	AttractorCRDName = "attractors.meridio.nordix.org"
	GatewayCRDName   = "gateways.meridio.nordix.org"
	VIPCRDName       = "vips.meridio.nordix.org"
)

type Framework struct {
	Namespace string
	Client    client.Client
	GinkgoT   GinkgoTInterface
}

type Operator struct {
	Namespeace                *corev1.Namespace
	Deployment                *appsv1.Deployment     // meridio-operator-controller-manager
	LeaderElectionRole        *rbacv1.Role           // meridio-operator-leader-election-role
	ManagerRole               *rbacv1.Role           // meridio-operator-manager-role
	LeaderElectionRoleBinding *rbacv1.RoleBinding    // meridio-operator-leader-election-rolebinding
	ManagerRoleBinding        *rbacv1.RoleBinding    // meridio-operator-manager-rolebinding
	ServiceAccount            *corev1.ServiceAccount // meridio-operator-controller-manager
	ConfigMap                 *corev1.ConfigMap      // meridio-operator-manager-config
	Service                   *corev1.Service        // meridio-operator-webhook-service
	// certificate.cert-manager.io/meridio-operator-serving-cert
	// issuer.cert-manager.io/meridio-operator-selfsigned-issuer
	// validatingwebhookconfiguration.admissionregistration.k8s.io/meridio-operator-validating-webhook-configuration
}

var namespace string
var mutating bool

func init() {
	flag.StringVar(&namespace, "namespace", "default", "specify the namespace for the tests to run")
	flag.BoolVar(&mutating, "mutating", true, "specify the namespace for the tests to run")
}

var fw = NewFramework()

// default trench used in all tests
func trench(namespace string) *meridiov1alpha1.Trench {
	return &meridiov1alpha1.Trench{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trenchName,
			Namespace: namespace,
		},
		Spec: meridiov1alpha1.TrenchSpec{
			IPFamily: "dualstack",
		},
	}
}

// default attractor used in all tests
func attractor(namespace string) *meridiov1alpha1.Attractor {
	return &meridiov1alpha1.Attractor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      attractorName,
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
			},
		},
		Spec: meridiov1alpha1.AttractorSpec{
			Gateways:   []string{"gateway-a", "gateway-b"},
			Vips:       []string{"vip-a", "vip-b"},
			Composites: []string{"conduit-a"},
			Interface: meridiov1alpha1.InterfaceSpec{
				Name:       "eth.100",
				PrefixIPv4: "169.254.100.0/24",
				PrefixIPv6: "100:100::/64",
				NSMVlan: meridiov1alpha1.NSMVlanSpec{
					VlanID:        pointer.Int32(100),
					BaseInterface: "eth0",
				},
			},
		},
	}
}

func conduit(namespace string) *meridiov1alpha1.Conduit {
	return &meridiov1alpha1.Conduit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "conduit-a",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
			},
		},
		Spec: meridiov1alpha1.ConduitSpec{
			Type: "stateless-lb",
		},
	}
}

// configmap name for the default trench
var configmapName = fmt.Sprintf("%s-%s", common.CMName, trenchName)

func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	fmt.Printf("The test suite will be running in %q namespace\n", namespace)
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	deployment := fw.GetOperator()
	Expect(deployment).ToNot(BeNil())
	Expect(fw.OperatorPodRestarts()).To(Equal(int32(0)))
	fw.tryCreateTrench()
})

var _ = AfterSuite(func() {
	Expect(fw.OperatorPodRestarts()).To(Equal(int32(0)))
})

func NewFramework() *Framework {
	t := GinkgoT()
	g := NewGomegaWithT(t)

	myScheme := runtime.NewScheme()

	g.Expect(kubescheme.AddToScheme(myScheme)).To(Succeed())
	g.Expect(scalescheme.AddToScheme(myScheme)).To(Succeed())
	g.Expect(apiextensionsv1.AddToScheme(myScheme)).To(Succeed())
	g.Expect(meridiov1alpha1.AddToScheme(myScheme)).To(Succeed())

	config := config.GetConfigOrDie()
	kubeAPIClient, err := client.New(config, client.Options{Scheme: myScheme})
	g.Expect(err).To(BeNil())

	return &Framework{
		Client:  kubeAPIClient,
		GinkgoT: t,
	}
}

func (fw *Framework) tryCreateTrench() {
	// test webhook connectivity by creating a trench
	trench := trench(namespace)

	Eventually(func(g Gomega) {
		fw.CleanUpTrenches()
		g.Expect(fw.CreateResource(trench)).Should(Succeed())
	}, timeout, interval).Should(Succeed())

	fw.CleanUpTrenches()
}

func (fw *Framework) GetOperator() *Operator {
	n := &corev1.Namespace{}
	Expect(fw.GetResource(client.ObjectKey{Name: namespace}, n)).To(Succeed())

	dep := &appsv1.Deployment{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-controller-manager",
	}, dep)).To(Succeed())

	svc := &corev1.Service{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-webhook-service",
	}, svc)).To(Succeed())

	sa := &corev1.ServiceAccount{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-controller-manager",
	}, sa)).To(Succeed())

	lr := &rbacv1.Role{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-leader-election-role",
	}, lr)).To(Succeed())

	mr := &rbacv1.Role{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-manager-role",
	}, mr)).To(Succeed())

	lrb := &rbacv1.RoleBinding{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-leader-election-rolebinding",
	}, lrb)).To(Succeed())

	mrb := &rbacv1.RoleBinding{}
	Expect(fw.GetResource(client.ObjectKey{
		Namespace: namespace,
		Name:      "meridio-operator-manager-rolebinding",
	}, mrb)).To(Succeed())

	//should have the AttractorCRDName CRD available in the cluster
	crd := &apiextensionsv1.CustomResourceDefinition{}
	Expect(fw.GetResource(client.ObjectKey{Name: AttractorCRDName}, crd)).To(Succeed())

	// should have the Gateway CRD available in the cluster
	crd = &apiextensionsv1.CustomResourceDefinition{}
	Expect(fw.GetResource(client.ObjectKey{Name: GatewayCRDName}, crd)).To(Succeed())

	// should have the VIP CRD available in the cluster
	crd = &apiextensionsv1.CustomResourceDefinition{}
	Expect(fw.GetResource(client.ObjectKey{Name: VIPCRDName}, crd)).To(Succeed())

	return &Operator{
		Namespeace:                n,
		Deployment:                dep,
		ServiceAccount:            sa,
		Service:                   svc,
		LeaderElectionRole:        lr,
		ManagerRole:               mr,
		LeaderElectionRoleBinding: lrb,
		ManagerRoleBinding:        mrb,
	}
}

func (fw *Framework) GetResource(key client.ObjectKey, obj client.Object) error {
	return fw.Client.Get(context.TODO(), key, obj)
}

func (fw *Framework) ListResources(obj client.ObjectList, opt ...client.ListOption) error {
	return fw.Client.List(context.TODO(), obj, opt...)
}

func (fw *Framework) DeleteResource(obj client.Object, opt ...client.DeleteOption) error {
	return fw.Client.Delete(context.TODO(), obj, opt...)
}

func (fw *Framework) DeleteAllOfResource(obj client.Object, opt ...client.DeleteAllOfOption) error {
	return fw.Client.DeleteAllOf(context.TODO(), obj, opt...)
}

func (fw *Framework) CreateResource(obj client.Object) error {
	return fw.Client.Create(context.TODO(), obj)
}

func (fw *Framework) UpdateResource(obj client.Object) error {
	return fw.Client.Update(context.TODO(), obj)
}

func (fw *Framework) OperatorPodRestarts() int32 {
	op := fw.GetOperator()
	label := op.Deployment.ObjectMeta.Labels
	pods := &corev1.PodList{}
	Expect(fw.ListResources(pods, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(label)})).To(Succeed())
	return pods.Items[0].Status.ContainerStatuses[0].RestartCount
}

func (fw *Framework) CleanUpTrenches() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Trench{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.TrenchList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())

}

func (fw *Framework) CleanUpAttractors() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Attractor{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.AttractorList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (fw *Framework) CleanUpVips() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Vip{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.VipList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (fw *Framework) CleanUpGateways() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Gateway{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.VipList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (fw *Framework) CleanUpConduits() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Conduit{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.ConduitList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (fw *Framework) CleanUpStreams() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Stream{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.StreamList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func (fw *Framework) CleanUpFlows() {
	Expect(fw.DeleteAllOfResource(&meridiov1alpha1.Flow{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: namespace}})).To(Succeed())
	Eventually(func() bool {
		lst := &meridiov1alpha1.FlowList{}
		err := fw.ListResources(lst, &client.ListOptions{Namespace: namespace})
		return err == nil && len(lst.Items) == 0
	}, 5*time.Second, interval).Should(BeTrue())
}

func AssertTrenchReady(trench *meridiov1alpha1.Trench) {
	ns := trench.ObjectMeta.Namespace
	name := trench.ObjectMeta.Name
	By("checking ipam StatefulSet")
	Eventually(func(g Gomega) {
		g.Expect(assertStatefulSetReady(strings.Join([]string{"ipam", name}, "-"), ns)).Should(Succeed())
	}, timeout, interval).Should(Succeed())

	By("checking nsp StatefulSet")
	Eventually(func(g Gomega) {
		g.Expect(assertStatefulSetReady(strings.Join([]string{"nsp", name}, "-"), ns)).Should(Succeed())
	}, timeout, interval).Should(Succeed())

	By("checking nsp service")
	nspServiceName := fmt.Sprintf("%s-%s", common.NspSvcName, name)
	service := &corev1.Service{}
	err := fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      nspServiceName,
	}, service)
	Expect(err).ToNot(HaveOccurred())
	Expect(service).ToNot(BeNil())

	By("checking ipam service")
	ipamServiceName := fmt.Sprintf("%s-%s", common.IpamSvcName, name)
	ipamService := &corev1.Service{}
	err = fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      ipamServiceName,
	}, ipamService)
	Expect(err).ToNot(HaveOccurred())
	Expect(service).ToNot(BeNil())

	By("checking role")
	roleName := fmt.Sprintf("%s-%s", common.RlName, name)
	role := &rbacv1.Role{}
	err = fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      roleName,
	}, role)
	Expect(err).ToNot(HaveOccurred())
	Expect(role).ToNot(BeNil())

	By("checking role binding")
	roleBindingName := fmt.Sprintf("%s-%s", common.RBName, name)
	roleBinding := &rbacv1.RoleBinding{}
	err = fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      roleBindingName,
	}, roleBinding)
	Expect(err).ToNot(HaveOccurred())
	Expect(roleBinding).ToNot(BeNil())

	By("checking service account")
	serviceAccountName := fmt.Sprintf("%s-%s", common.SAName, name)
	serviceAccount := &corev1.ServiceAccount{}
	err = fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      serviceAccountName,
	}, serviceAccount)
	Expect(err).ToNot(HaveOccurred())
	Expect(serviceAccount).ToNot(BeNil())
}

func AssertAttractorReady(attractor *meridiov1alpha1.Attractor) {
	ns := attractor.ObjectMeta.Namespace

	By("checking nse vlan deployment")
	Eventually(func(g Gomega) {
		g.Expect(assertDeploymentReady(strings.Join([]string{"nse-vlan", attractor.ObjectMeta.Name}, "-"), ns)).Should(Succeed())
	}, timeout, interval).Should(Succeed())

	// By("checking lb-fe deployment")
	// Eventually(func(g Gomega) {
	// 	g.Expect(assertDeploymentReady(strings.Join([]string{"lb-fe", attractor.ObjectMeta.Name}, "-"), ns)).Should(Succeed())
	// }, timeout, interval).Should(Succeed())
}

func AssertConduitReady(conduit *meridiov1alpha1.Conduit) {
	// ns := conduit.ObjectMeta.Namespace

	// By("checking proxy deployment")
	// Eventually(func(g Gomega) {
	// 	g.Expect(assertDaemonsetReady(strings.Join([]string{"proxy", trenchName}, "-"), ns)).Should(Succeed())
	// }, timeout, interval).Should(Succeed())
}

func AssertMeridioDeploymentsReady(trench *meridiov1alpha1.Trench,
	attractor *meridiov1alpha1.Attractor,
	conduit *meridiov1alpha1.Conduit) {
	AssertTrenchReady(trench)
	AssertAttractorReady(attractor)
	AssertConduitReady(conduit)
}

func assertDeploymentReady(name, ns string) error {
	dep := &appsv1.Deployment{}
	// checking if the deployment exists
	err := fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, dep)
	if err != nil {
		return err
	}

	// checking all replicas are ready
	if dep.Status.ReadyReplicas != dep.Status.Replicas {
		return fmt.Errorf("Status.ReadyReplicas not equal Status.Replicas")
	}

	// checking all pods are ready and never restarted
	listOptions := &client.ListOptions{
		LabelSelector: labels.Set(dep.Labels).AsSelector(),
	}
	return podsRunning(listOptions)
}

func assertStatefulSetReady(name, ns string) error {
	dep := &appsv1.StatefulSet{}
	// checking if the deployment exists
	err := fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, dep)
	if err != nil {
		return err
	}

	// checking all replicas are ready
	if dep.Status.ReadyReplicas != dep.Status.Replicas {
		return fmt.Errorf("Status.ReadyReplicas not equal Status.Replicas")
	}

	// checking all pods are ready and never restarted
	listOptions := &client.ListOptions{
		LabelSelector: labels.Set(dep.Labels).AsSelector(),
	}
	return podsRunning(listOptions)
}

func assertDaemonsetReady(name, ns string) error {
	ds := &appsv1.DaemonSet{}
	// checking if the daemonset exists
	err := fw.GetResource(client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, ds)
	if err != nil {
		return err
	}

	// checking all desired replicas are ready"
	if ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
		return fmt.Errorf("Status.NumberReady not equal Status.DesiredNumberScheduled")
	}
	listOptions := &client.ListOptions{
		LabelSelector: labels.Set(ds.Labels).AsSelector(),
	}
	return podsRunning(listOptions)
}

func podsRunning(opts client.ListOption) error {
	pods := &corev1.PodList{}
	err := fw.Client.List(context.Background(), pods, opts)
	if err != nil {
		return err
	}
	// wait for all the pods of the deployment are in running status
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pod %s is not running", pod.ObjectMeta.Name)
		}
	}
	// check the restart count of each container of each pod
	for _, pod := range pods.Items {
		for _, container := range pod.Status.ContainerStatuses {
			if container.RestartCount != int32(0) {
				return fmt.Errorf("container %s restart is not 0", container.Name)
			}
		}
	}
	return nil
}
