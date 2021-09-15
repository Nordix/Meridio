package e2e

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Attractor", func() {
	trench := &meridiov1alpha1.Trench{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trenchName,
			Namespace: namespace,
		},
		Spec: meridiov1alpha1.TrenchSpec{
			IPFamily: "DualStack",
		},
	}
	attractor := &meridiov1alpha1.Attractor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      attractorName,
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
			},
		},
		Spec: meridiov1alpha1.AttractorSpec{
			VlanID:         100,
			VlanInterface:  "eth0",
			Replicas:       replicas, // replica of lb-fe
			Gateways:       []string{"gateway-a", "gateway-b"},
			Vips:           []string{"vip-a", "vip-b"},
			VlanPrefixIPv4: "169.254.100.0/24",
			VlanPrefixIPv6: "100:100::/64",
		},
	}
	Context("When creating an attractor", func() {
		BeforeEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
		})
		// operator scope
		Context("in another namespace than the trench and operator", func() {
			another := "another"
			nsanother := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: another},
			}
			BeforeEach(func() {
				// Deep copy to avoid original variables to be overwritten
				Expect(fw.CreateResource(trench.DeepCopy())).Should(Succeed())
				Expect(fw.CreateResource(nsanother)).Should(Succeed())
			})

			AfterEach(func() {
				Expect(fw.DeleteResource(nsanother)).Should(Succeed())
				fw.CleanUpTrenches()
			})

			It("will not have any updates", func() {
				attr := attractor.DeepCopy()
				attr.Namespace = another
				Expect(fw.CreateResource(attr)).Should(Succeed())

				By("checking the status is empty")
				assertAttractorStatus(attractor, meridiov1alpha1.NoPhase)

				By("checking no attractor resources are created")
				assertAttractorResourcesNotExist()
			})
		})
		// attractor controller behavior
		Context("without a trench", func() {
			attr := &meridiov1alpha1.Attractor{}

			BeforeEach(func() {
				// Deep copy to avoid original variables to be overwritten
				Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			})

			It("will create a non-functioning attractor", func() {
				By("checking the existence of attactor")
				err := fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
				Expect(err).Should(BeNil())
				Expect(attr).ShouldNot(BeNil())

				By("checking status being disengaged")
				assertAttractorStatus(attractor, meridiov1alpha1.Disengaged)

				By("checking resources are created")
				assertAttractorResourcesNotExist()
			})

		})

		Context("with a trench", func() {
			BeforeEach(func() {
				// Deep copy to avoid original variables to be overwritten
				Expect(fw.CreateResource(trench.DeepCopy())).Should(Succeed())
				Expect(fw.CreateResource(attractor.DeepCopy())).Should(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpAttractors()
			})

			It("will create a functioning attractor", func() {
				attr := &meridiov1alpha1.Attractor{}

				By("checking the existence of attractor")
				err := fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
				Expect(err).Should(BeNil())
				Expect(attr).ShouldNot(BeNil())

				By("checking status being engaged")
				assertAttractorStatus(attractor, meridiov1alpha1.Engaged)

				By("checking if lb-fe pods are in running state")
				loadBalancerName := fmt.Sprintf("%s-%s", common.LBName, trench.ObjectMeta.Name)
				Eventually(func() int32 {
					deployment, _ := fw.Clientset.AppsV1().Deployments(namespace).Get(context.Background(), loadBalancerName, metav1.GetOptions{})
					return deployment.Status.Replicas
				}, timeout, interval).Should(Equal(*replicas))

				By("checking if nse-vlan pods are in running state")
				nseVLANName := fmt.Sprintf("%s-%s", common.NseName, attractorName)
				Eventually(func() int32 {
					deployment, _ := fw.Clientset.AppsV1().Deployments(namespace).Get(context.Background(), nseVLANName, metav1.GetOptions{})
					return deployment.Status.Replicas
				}, timeout, interval).Should(Equal(*replicas))

				By("checking if configmap has been created")
				configmapName := fmt.Sprintf("%s-%s", common.CMName, trench.ObjectMeta.Name)
				configmap := &corev1.ConfigMap{}
				Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: attractor.Namespace}, configmap)).To(Succeed())
				Expect(configmap).ToNot(BeNil())
			})

			It("can update the replicas of the lb-fe", func() {
				attr := &meridiov1alpha1.Attractor{}

				By("updating attractor spec.replicas")
				Eventually(func(g Gomega) {
					err := fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
					g.Expect(err).ToNot(HaveOccurred())
					*attr.Spec.Replicas = 4
					g.Expect(fw.UpdateResource(attr)).To(Succeed())
				}, timeout, interval).Should(Succeed())

				By("checking status being engaged")
				assertAttractorStatus(attractor, meridiov1alpha1.Engaged)

				By("checking the lb-fe replicas")
				Eventually(func() int32 {
					deployment := &appsv1.Deployment{}
					loadBalancerName := fmt.Sprintf("%s-%s", common.LBName, trench.ObjectMeta.Name)
					Expect(fw.GetResource(client.ObjectKey{Name: loadBalancerName, Namespace: namespace}, deployment)).To(Succeed())
					return deployment.Status.Replicas
				}, timeout, interval).Should(Equal(*attr.Spec.Replicas))
			})
		})
	})

	Context("When deleting an attractor", func() {
		BeforeEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
			// Deep copy to avoid original variables to be overwritten
			Expect(fw.CreateResource(trench.DeepCopy())).Should(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).Should(Succeed())
		})

		It("deletes attractor resources by deleting attractor", func() {
			attr := &meridiov1alpha1.Attractor{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr)).To(Succeed())
			Expect(fw.DeleteResource(attr)).Should(Succeed())

			By("checking attractor resources")
			assertAttractorResourcesNotExist()
		})

		It("deletes attractor resources by deleting trench", func() {
			tr := &meridiov1alpha1.Trench{}
			err := fw.GetResource(client.ObjectKeyFromObject(trench), tr)
			Expect(err).ToNot(HaveOccurred())
			Expect(fw.DeleteResource(tr)).Should(Succeed())

			By("checking attractor resources")
			assertAttractorResourcesNotExist()
		})
	})
})

func assertAttractorResourcesNotExist() {
	By("checking there is no load balancer deployments")
	loadBalancerName := fmt.Sprintf("%s-%s", common.LBName, trenchName)
	Eventually(func() bool {
		err := fw.GetResource(client.ObjectKey{Name: loadBalancerName, Namespace: namespace}, &appsv1.Deployment{})
		return err != nil && apierrors.IsNotFound(err)
	}, 5*time.Second).Should(BeTrue())

	By("checking there is no nse-vlan deployments")
	nseVLANName := fmt.Sprintf("%s-%s", common.NseName, attractorName)
	Eventually(func() bool {
		err := fw.GetResource(client.ObjectKey{Name: nseVLANName, Namespace: namespace}, &appsv1.Deployment{})
		return err != nil && apierrors.IsNotFound(err)
	}, 5*time.Second).Should(BeTrue())

	By("checking there is no configmap")
	configmapName := fmt.Sprintf("%s-%s", common.CMName, trenchName)
	Eventually(func() bool {
		err := fw.GetResource(client.ObjectKey{Namespace: namespace, Name: configmapName}, &corev1.ConfigMap{})
		return err != nil && apierrors.IsNotFound(err)
	}, 5*time.Second).Should(BeTrue())
}

func assertAttractorStatus(attractor *meridiov1alpha1.Attractor, status meridiov1alpha1.ConfigStatus) {
	attr := &meridiov1alpha1.Attractor{}
	Eventually(func() meridiov1alpha1.ConfigStatus {
		fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
		return attr.Status.LbFe
	}, 5*time.Second, interval).Should(Equal(status))
}
