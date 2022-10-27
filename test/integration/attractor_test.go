package e2e

import (
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"github.com/nordix/meridio-operator/testdata/utils"
	"github.com/nordix/meridio/pkg/configuration/reader"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Attractor", func() {
	trench := trench(namespace)
	attractor := attractor(namespace)

	When("creating an attractor", func() {
		BeforeEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
			// wait for the old instances to be deleted
			time.Sleep(time.Second)
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

			It("will be created but create no child resources", func() {
				attr := attractor.DeepCopy()
				attr.Namespace = another
				Expect(fw.CreateResource(attr)).Should(Succeed())

				By("checking no attractor resources are created")
				assertAttractorResourcesNotExist(attr)
			})
		})

		// attractor controller behavior
		Context("without a trench", func() {
			It("will fail in creation", func() {
				Expect(fw.CreateResource(attractor.DeepCopy())).ToNot(Succeed())

				By("checking the existence of attactor")
				err := fw.GetResource(client.ObjectKeyFromObject(attractor), &meridiov1alpha1.Attractor{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())

				By("checking no child resources are created")
				assertAttractorResourcesNotExist(attractor)
			})
		})

		Context("with a trench", func() {
			BeforeEach(func() {
				fw.CreateResource(trench.DeepCopy())
			})

			AfterEach(func() {
				fw.CleanUpAttractors()
				// wait for the old instances to be deleted
				time.Sleep(time.Second)
			})

			When("missing parameters", func() {
				It("will not be created ", func() {
					testAttractor := attractor.DeepCopy()
					testAttractor.Spec.Interface.NSMVlan = meridiov1alpha1.NSMVlanSpec{}

					err := fw.CreateResource(testAttractor)
					Expect(err).ToNot(BeNil())

					By("checking no attractor resources are created")
					assertAttractorResourcesNotExist(testAttractor)
				})
			})

			When("with a valid attractor", func() {
				It("will create a functioning attractor", func() {
					testAttractor := attractor.DeepCopy()
					Expect(fw.CreateResource(testAttractor)).To(Succeed())
					attr := &meridiov1alpha1.Attractor{}

					By("checking the existence of attractor")
					err := fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
					Expect(err).Should(BeNil())
					Expect(attr).ShouldNot(BeNil())

					By("checking configmap has this item")
					assertAttractorItemInConfigMap(attr, configmapName, true)

					By("checking if attractor's child resources are in running state")
					AssertAttractorReady(attr)
				})

				It("will fail creating the second attractor", func() {
					testAttractor := attractor.DeepCopy()
					Expect(fw.CreateResource(testAttractor)).To(Succeed())
					attractorB := &meridiov1alpha1.Attractor{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "attractor-b",
							Namespace: namespace,
							Labels: map[string]string{
								"trench": trenchName,
							},
						},
						Spec: meridiov1alpha1.AttractorSpec{
							Interface: meridiov1alpha1.InterfaceSpec{
								Name:       "nsm-vlan1",
								Type:       "nsm-vlan",
								PrefixIPv4: "169.254.100.0/24",
								PrefixIPv6: "100:100::/64",
								NSMVlan: meridiov1alpha1.NSMVlanSpec{
									VlanID:        pointer.Int32(100),
									BaseInterface: "eth0",
								},
							},
						},
					}

					Expect(fw.CreateResource(attractorB)).ToNot(Succeed())

					By("checking configmap has this item")
					assertAttractorItemInConfigMap(attractorB, configmapName, false)

					By("checking the existence of attactor")
					err := fw.GetResource(client.ObjectKeyFromObject(attractorB), &meridiov1alpha1.Attractor{})
					Expect(apierrors.IsNotFound(err)).To(BeTrue())
				})
			})
		})

		Context("with two trenches", func() {
			trenchB := &meridiov1alpha1.Trench{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "trench-b",
					Namespace: namespace,
				},
				Spec: meridiov1alpha1.TrenchSpec{
					IPFamily: string(meridiov1alpha1.IPv4),
				},
			}
			attractorB := attractor.DeepCopy()
			attractorB.Labels["trench"] = "trench-b"
			attractorB.Name = "attractor-b"
			BeforeEach(func() {
				fw.CreateResource(trench.DeepCopy())
				fw.CreateResource(trenchB.DeepCopy())
			})

			AfterEach(func() {
				fw.CleanUpAttractors()
				// wait for the old instances to be deleted
				time.Sleep(time.Second)
			})

			It("controlls resources respectively", func() {
				// create attractor a
				Expect(fw.CreateResource(attractor.DeepCopy())).Should(Succeed())

				By("checking resources created by attractor a")
				AssertAttractorReady(attractor)
				assertAttractorItemInConfigMap(attractor, configmapName, true)

				// create attractor b
				Expect(fw.CreateResource(attractorB)).Should(Succeed())
				By("checking resources created by attractor b")
				AssertAttractorReady(attractorB)
				assertAttractorItemInConfigMap(attractorB, common.ConfigMapName(trenchB), true)

				// delete attractor b
				Expect(fw.DeleteResource(attractorB)).Should(Succeed())
				By("checking attractor a is not impacted by attractor b")
				AssertAttractorReady(attractor)
				assertAttractorItemInConfigMap(attractor, configmapName, true)

				By("checking attractor b is handled correctly")
				assertAttractorResourcesNotExist(attractorB)
				assertAttractorItemInConfigMap(attractorB, common.ConfigMapName(trenchB), false)
			})
		})
	})

	When("updating", func() {
		BeforeEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
			// Deep copy to avoid original variables to be overwritten
			Expect(fw.CreateResource(trench.DeepCopy())).Should(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).Should(Succeed())
			AssertAttractorReady(attractor)
		})

		It("can update the gateways and vips of lb-fe", func() {
			attr := &meridiov1alpha1.Attractor{}

			By("updating attractor spec.gateways and spec.vips")
			Eventually(func(g Gomega) {
				err := fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
				g.Expect(err).ToNot(HaveOccurred())
				attr.Spec.Gateways = []string{"gateway1"}
				attr.Spec.Vips = []string{"vip1"}
				g.Expect(fw.UpdateResource(attr)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("checking the configmap")
			assertAttractorItemInConfigMap(attractor, configmapName, false)
			assertAttractorItemInConfigMap(attr, configmapName, true)
		})

		It("can update the replicas of the lb-fe", func() {
			attr := &meridiov1alpha1.Attractor{}
			By("checking current replica is 1")
			deployment := &appsv1.Deployment{}
			loadBalancerName := fmt.Sprintf("%s-%s", common.LBName, attractorName)
			Expect(fw.GetResource(client.ObjectKey{Name: loadBalancerName, Namespace: namespace}, deployment)).To(Succeed())
			Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))

			By("updating attractor spec.replicas to be 2")
			Eventually(func(g Gomega) {
				err := fw.GetResource(client.ObjectKeyFromObject(attractor), attr)
				g.Expect(err).ToNot(HaveOccurred())
				*attr.Spec.Replicas = 2
				g.Expect(fw.UpdateResource(attr)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			By("checking the lb-fe replicas to be 2")
			Eventually(func() int32 {
				Expect(fw.GetResource(client.ObjectKey{Name: loadBalancerName, Namespace: namespace}, deployment)).To(Succeed())
				By(fmt.Sprintf("current replicas: %v", *deployment.Spec.Replicas))
				return *deployment.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(2)))
		})
	})

	When("deleting an attractor", func() {
		BeforeEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
			// Deep copy to avoid original variables to be overwritten
			fw.CreateResource(trench.DeepCopy())
			fw.CreateResource(attractor.DeepCopy())
			time.Sleep(time.Second)
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
		})

		It("deletes attractor resources by deleting itself", func() {
			attr := &meridiov1alpha1.Attractor{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr)).To(Succeed())
			Expect(fw.DeleteResource(attr)).Should(Succeed())

			By("checking attractor resources")
			assertAttractorResourcesNotExist(attractor)
			assertAttractorItemInConfigMap(attr, configmapName, false)
		})

		It("deletes attractor resources by deleting trench", func() {
			tr := &meridiov1alpha1.Trench{}
			err := fw.GetResource(client.ObjectKeyFromObject(trench), tr)
			Expect(err).ToNot(HaveOccurred())
			Expect(fw.DeleteResource(tr)).Should(Succeed())

			By("checking attractor resources")
			assertAttractorResourcesNotExist(attractor)
		})
	})
})

func assertAttractorResourcesNotExist(attr *meridiov1alpha1.Attractor) {
	namespace := attr.ObjectMeta.Namespace
	By("checking there is no load balancer deployments")
	Eventually(func() bool {
		err := fw.GetResource(client.ObjectKey{
			Name:      common.LbFeDeploymentName(attr),
			Namespace: attr.ObjectMeta.Namespace,
		}, &appsv1.Deployment{})
		return err != nil && apierrors.IsNotFound(err)
	}, 5*time.Second).Should(BeTrue())

	By("checking there is no nse-vlan deployments")
	nseVLANName := common.NSEDeploymentName(attr)
	Eventually(func(g Gomega) {
		g.Expect(fw.GetResource(client.ObjectKey{
			Name:      nseVLANName,
			Namespace: namespace,
		}, &appsv1.Deployment{})).ToNot(Succeed())
	}, 5*time.Second).Should(Succeed())
}

func assertAttractorItemInConfigMap(attr *meridiov1alpha1.Attractor, configmapName string, in bool) {
	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	configmap := &corev1.ConfigMap{}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, vip key has an item same as vip resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: attr.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())

		lst, err := reader.UnmarshalAttractors(configmap.Data[reader.AttractorsConfigKey])
		g.Expect(err).To(BeNil())

		mp := utils.MakeMapFromAttractorList(lst)
		a, ok := mp[attr.ObjectMeta.Name]

		// then checking in configmap data, vip key has an item same as vip resource
		equal := equality.Semantic.DeepEqual(a, reader.Attractor{
			Name:     attr.ObjectMeta.Name,
			Vips:     attr.Spec.Vips,
			Gateways: attr.Spec.Gateways,
			Trench:   attr.ObjectMeta.Labels["trench"]})
		return ok && equal
	}, timeout).Should(matcher)
}
