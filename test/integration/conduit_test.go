package e2e

import (
	"time"

	meridiov1alpha1 "github.com/nordix/meridio/api/v1alpha1"
	config "github.com/nordix/meridio/pkg/configuration/reader"
	"github.com/nordix/meridio/pkg/controllers/common"
	"github.com/nordix/meridio/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Conduit", func() {
	trench := trench(namespace)
	attractor := attractor(namespace)
	conduit := conduit(namespace)

	BeforeEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpConduits()
		// wait for the old instances to be deleted
		time.Sleep(time.Second)
	})

	Context("When creating a conduit", func() {
		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
			fw.CleanUpConduits()
		})

		Context("without a trench", func() {
			It("will failed in creation", func() {
				Expect(fw.CreateResource(conduit.DeepCopy())).ToNot(Succeed())

				By("checking it does not exist")
				err := fw.GetResource(client.ObjectKeyFromObject(conduit), &meridiov1alpha1.Conduit{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		// conduit must be created after attractor
		Context("without an attractor", func() {
			It("will failed in creation", func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(conduit.DeepCopy())).ToNot(Succeed())

				By("checking it does not exist")
				err := fw.GetResource(client.ObjectKeyFromObject(conduit), &meridiov1alpha1.Conduit{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("with trench and attractor", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			})
			JustBeforeEach(func() {
				Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpTrenches()
				fw.CleanUpAttractors()
				fw.CleanUpConduits()
			})

			It("will be created successfully", func() {
				By("checking if the conduit exists")
				con := &meridiov1alpha1.Conduit{}
				err := fw.GetResource(client.ObjectKeyFromObject(conduit), con)
				Expect(err).NotTo(HaveOccurred())
				Expect(con).NotTo(BeNil())

				By("checking the deployment is ready")
				AssertConduitReady(conduit)

				By("checking conduit is in configmap data")
				assertConduitItemInConfigMap(conduit, configmapName, true)
			})
		})

		Context("two trenches", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(
					&meridiov1alpha1.Trench{
						ObjectMeta: v1.ObjectMeta{
							Name:      "trench-b",
							Namespace: namespace,
						},
						Spec: meridiov1alpha1.TrenchSpec{
							IPFamily: string(meridiov1alpha1.IPv4),
						},
					})).To(Succeed())
			})

			It("needs attractor respectively to create conduits for each trench", func() {
				By("create attractor for trench-a")
				Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())

				By("create conduit for trench-a")
				Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
				assertConduitItemInConfigMap(conduit, configmapName, true)

				By("create conduit will fail without attractor for trench-b")
				conduitB := &meridiov1alpha1.Conduit{
					ObjectMeta: v1.ObjectMeta{
						Name:      "conduit-b",
						Namespace: namespace,
						Labels: map[string]string{
							"trench": "trench-b",
						},
					},
					Spec: meridiov1alpha1.ConduitSpec{
						Type: "stateless-lb",
					},
				}
				Expect(fw.CreateResource(conduitB)).ToNot(Succeed())
				assertConduitItemInConfigMap(conduitB, common.CMName+"-trench-b", false)

				By("create attractor for trench-b")
				attractorB := attractor.DeepCopy()
				attractorB.ObjectMeta.Name = "attractor-b"
				attractorB.Labels["trench"] = "trench-b"
				Expect(fw.CreateResource(attractorB.DeepCopy())).To(Succeed())

				By("create conduit for trench-b")
				Expect(fw.CreateResource(conduitB)).To(Succeed())
				assertConduitItemInConfigMap(conduitB, configmapName, false)
				assertConduitItemInConfigMap(conduitB, common.CMName+"-trench-b", true)
			})
		})
	})

	Context("When deleting a conduit", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			assertConduitItemInConfigMap(conduit, configmapName, true)
		})

		It("will update configmap", func() {
			con := &meridiov1alpha1.Conduit{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(conduit), con)).To(Succeed())
			Expect(fw.DeleteResource(con)).To(Succeed())

			By("checking configmap")
			assertConduitItemInConfigMap(con, configmapName, false)
		})
	})

	Context("When deleting", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			AssertConduitReady(conduit)
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpConduits()
			fw.CleanUpAttractors()
		})

		It("will be deleted by deleting the trench", func() {
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(trench), tr)).To(Succeed())
			Expect(fw.DeleteResource(tr)).To(Succeed())

			By("checking if conduit exists")
			Eventually(func() bool {
				s := &meridiov1alpha1.Conduit{}
				err := fw.GetResource(client.ObjectKeyFromObject(conduit), s)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})

		It("will be deleted by deleting itself", func() {
			c := &meridiov1alpha1.Conduit{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(conduit), c)).To(Succeed())
			Expect(fw.DeleteResource(c)).To(Succeed())

			By("checking if conduit exists")
			Eventually(func() bool {
				s := &meridiov1alpha1.Conduit{}
				err := fw.GetResource(client.ObjectKeyFromObject(conduit), s)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())

			By("checking the conduit is deleted from configmap")
			assertConduitItemInConfigMap(conduit, configmapName, false)
		})
	})

	Context("when updating the configmap directly", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			assertConduitItemInConfigMap(conduit, configmapName, true)
		})
		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpConduits()
			fw.CleanUpAttractors()
		})
		It("will be reverted according to the current conduit", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: conduit.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking conduit item still in the configmap")
			assertConduitItemInConfigMap(conduit, configmapName, true)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: conduit.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[config.ConduitsConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking conduit item still in the configmap")
			assertConduitItemInConfigMap(conduit, configmapName, true)
		})
	})
})

func assertConduitItemInConfigMap(con *meridiov1alpha1.Conduit, configmapName string, in bool) {
	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	configmap := &corev1.ConfigMap{}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, conduit key has an item same as conduit resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: con.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())

		conduitsconfig, err := config.UnmarshalConduits(configmap.Data[config.ConduitsConfigKey])
		g.Expect(err).To(BeNil())

		conduitmap := utils.MakeMapFromConduitList(conduitsconfig)
		conduitInConfig, ok := conduitmap[con.ObjectMeta.Name]

		// then checking in configmap data, conduit key has an item same as conduit resource
		equal := equality.Semantic.DeepEqual(conduitInConfig, config.Conduit{
			Name:   con.ObjectMeta.Name,
			Trench: con.ObjectMeta.Labels["trench"]})
		return ok && equal
	}, timeout).Should(matcher)
}
