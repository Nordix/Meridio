package e2e

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"github.com/nordix/meridio-operator/controllers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Vip", func() {
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
	vipA := &meridiov1alpha1.Vip{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vip-a",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
			},
		},
		Spec: meridiov1alpha1.VipSpec{
			Address: "10.0.0.0/28",
		},
	}
	configmapName := fmt.Sprintf("%s-%s", common.CMName, trench.ObjectMeta.Name)

	BeforeEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpVips()
	})

	AfterEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpVips()
	})

	Context("When creating a vip", func() {
		AfterEach(func() {
			fw.CleanUpVips()
		})
		Context("without a trench", func() {
			It("will be created with disengaged status", func() {
				Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())

				By("checking the existence")
				vp := &meridiov1alpha1.Vip{}
				fw.GetResource(client.ObjectKeyFromObject(vipA), vp)
				Expect(vp).NotTo(BeNil())

				By("checking the status to be disengaged")
				assertVipStatus(vipA, meridiov1alpha1.Disengaged)
			})
		})

		Context("with trench", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			})
			JustBeforeEach(func() {
				Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpTrenches()
				fw.CleanUpVips()
			})

			It("will be created with engaged status", func() {
				By("checking if the vip exists")
				vp := &meridiov1alpha1.Vip{}
				fw.GetResource(client.ObjectKeyFromObject(vipA), vp)
				Expect(vp).NotTo(BeNil())

				By("checking the status to be engaged")
				assertVipStatus(vipA, meridiov1alpha1.Engaged)
			})

			Context("with attractor expecting to use this vip", func() {
				BeforeEach(func() {
					Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
				})
				AfterEach(func() {
					fw.CleanUpAttractors()
				})

				It("will be collected into the configmap", func() {
					By("checking vip is in configmap data")
					vipItemInConfigMap(vipA, configmapName)
				})

				It("will update the vip-in-use status in attractor", func() {
					attractorVipInUseEqualTo(attractor, vipA.ObjectMeta.Name)
				})

				It("will remove vip if attractor removes it from the expected vip list", func() {
					By("updating the attractor")
					Eventually(func(g Gomega) {
						attr := &meridiov1alpha1.Attractor{}
						g.Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr))
						attr.Spec.Vips = []string{}
						g.Expect(fw.UpdateResource(attr)).To(Succeed())
					}, timeout, interval).Should(Succeed())

					By("checking configmap doesn't have such item")
					vipItemNotInConfigMap(vipA, configmapName)

					By("checking vip-in-use status in attractor")
					attractorVipInUseEqualTo(attractor)
				})
			})

			Context("with attractor not expecting to use this vip", func() {
				BeforeEach(func() {
					newAttr := attractor.DeepCopy()
					newAttr.Spec.Vips = []string{}
					Expect(fw.CreateResource(newAttr)).To(Succeed())
				})
				AfterEach(func() {
					fw.CleanUpAttractors()
				})

				It("will not be collected into the configmap", func() {
					By("checking configmap doesn't have such item")
					vipItemNotInConfigMap(vipA, configmapName)
				})

				It("will be collected if the vip in attractor is updated", func() {
					By("updating the attractor")
					Eventually(func(g Gomega) {
						attr := &meridiov1alpha1.Attractor{}
						g.Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr))
						attr.Spec.Vips = []string{vipA.ObjectMeta.Name}
						g.Expect(fw.UpdateResource(attr)).To(Succeed())
					}).Should(Succeed())

					By("checking vip is in configmap")
					vipItemInConfigMap(vipA, configmapName)
				})
			})
		})
	})

	Context("When deleting a vip", func() {
		vp := vipA.DeepCopy()

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(vp)).To(Succeed())
			vipItemInConfigMap(vp, configmapName)
		})
		JustBeforeEach(func() {
			Expect(fw.DeleteResource(vp)).To(Succeed())
		})
		It("will update configmap and attractor", func() {
			By("checking configmap")
			vipItemNotInConfigMap(vp, configmapName)

			By("checking vip-in-use status in attractor")
			attractorVipInUseEqualTo(attractor)
		})
	})

	Context("When updating a vip", func() {
		Context("with attractor expecting to use it", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())
			})
			It("updates the configmap and attractor", func() {
				var vp = &meridiov1alpha1.Vip{}
				Eventually(func(g Gomega) {
					g.Expect(fw.GetResource(client.ObjectKeyFromObject(vipA), vp)).To(Succeed())
					vp.Spec.Address = "20.0.0.0/32"
					g.Expect(fw.UpdateResource(vp)).To(Succeed())
				}).Should(Succeed())

				By("checking configmap")
				vipItemInConfigMap(vp, configmapName)

				By("checking attractor")
				attractorVipInUseEqualTo(attractor, vp.ObjectMeta.Name)
			})
		})
	})

	Context("checking meridio pods", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			fw.AssertTrenchReady(trench)
			fw.AssertAttractorReady(trench, attractor)
		})
		It("will not trigger restarts in any of the meridio pods", func() {
			Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())

			By("Checking the restarts of meridio pods")
			fw.AssertTrenchReady(trench)
			fw.AssertAttractorReady(trench, attractor)
		})
	})

	Context("When deleting a trench", func() {
		vp := vipA.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(vp)).To(Succeed())
			assertVipStatus(vipA, meridiov1alpha1.Engaged)
		})
		It("will be deleted", func() {
			Expect(fw.DeleteResource(tr)).To(Succeed())
			By("checking if vip exists")
			Eventually(func() bool {
				vp := &meridiov1alpha1.Vip{}
				err := fw.GetResource(client.ObjectKeyFromObject(vipA), vp)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})
	})

	Context("when updating the configmap directly", func() {
		vp := vipA.DeepCopy()
		attr := attractor.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(attr)).To(Succeed())
			assertAttractorStatus(attractor, meridiov1alpha1.Engaged)
			Expect(fw.CreateResource(vp)).To(Succeed())
			assertVipStatus(vipA, meridiov1alpha1.Engaged)
			vipItemInConfigMap(vp, configmapName)
		})
		It("will be reverted according to the current vip", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vp.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking vip item still in the configmap")
			vipItemInConfigMap(vp, configmapName)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vp.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[config.VipsConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking vip item still in the configmap")
			vipItemInConfigMap(vp, configmapName)
		})
	})
})

func vipItemNotInConfigMap(vip *meridiov1alpha1.Vip, configmapName string) {
	configmap := &corev1.ConfigMap{}
	var ok bool
	Eventually(func(g Gomega) {
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vip.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())
		vipsconfig, err := config.UnmarshalVipConfig(configmap.Data[config.VipsConfigKey])
		g.Expect(err).To(BeNil())

		vipmap := config.MakeMapFromVipList(vipsconfig)
		_, ok = vipmap[vip.ObjectMeta.Name]
		g.Expect(ok).To(BeFalse())
	}, timeout).Should(Succeed())
}

func vipItemInConfigMap(vip *meridiov1alpha1.Vip, configmapName string) {
	configmap := &corev1.ConfigMap{}
	var vipInConfig config.Vip
	var ok bool
	Eventually(func(g Gomega) {
		// checking in configmap data, vip key has an item same as vip resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vip.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())
		vipsconfig, err := config.UnmarshalVipConfig(configmap.Data[config.VipsConfigKey])
		g.Expect(err).To(BeNil())

		vipmap := config.MakeMapFromVipList(vipsconfig)
		vipInConfig, ok = vipmap[vip.ObjectMeta.Name]
		g.Expect(ok).To(BeTrue())

		// then checking in configmap data, vip key has an item same as vip resource
		g.Expect(vipInConfig).To(Equal(config.Vip{Name: vip.ObjectMeta.Name, Address: vip.Spec.Address}))
	}, timeout).Should(Succeed())

}

func attractorVipInUseEqualTo(attractor *meridiov1alpha1.Attractor, vpNames ...string) {
	Eventually(func(g Gomega) {
		attr := &meridiov1alpha1.Attractor{}
		g.Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr)).To(Succeed())
		g.Expect(attr.Status.VipsInUse).To(Equal(vpNames))
	}, timeout).Should(Succeed())
}

func assertVipStatus(vip *meridiov1alpha1.Vip, status meridiov1alpha1.ConfigStatus) {
	vp := &meridiov1alpha1.Vip{}
	Eventually(func() meridiov1alpha1.ConfigStatus {
		fw.GetResource(client.ObjectKeyFromObject(vip), vp)
		return vp.Status.Status
	}).Should(Equal(status))
}
