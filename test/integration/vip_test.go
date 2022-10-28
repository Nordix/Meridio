package e2e

import (
	"fmt"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"github.com/nordix/meridio-operator/testdata/utils"
	config "github.com/nordix/meridio/pkg/configuration/reader"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Vip", func() {
	trench := trench(namespace)
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
		// wait for the old instances to be deleted
		time.Sleep(time.Second)
	})

	Context("When creating a vip", func() {
		vipB := &meridiov1alpha1.Vip{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vip-b",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchName,
				},
			},
			Spec: meridiov1alpha1.VipSpec{
				Address: "20.0.0.0/28",
			},
		}

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpVips()
		})
		Context("without a trench", func() {
			It("will fail in creation", func() {
				Expect(fw.CreateResource(vipA.DeepCopy())).ToNot(Succeed())

				By("checking the existence")
				err := fw.GetResource(client.ObjectKeyFromObject(vipA), &meridiov1alpha1.Vip{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("with one trench", func() {
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

			It("will be created successfully", func() {
				By("checking if the vip exists")
				vp := &meridiov1alpha1.Vip{}
				err := fw.GetResource(client.ObjectKeyFromObject(vipA), vp)
				Expect(err).To(BeNil())
				Expect(vp).NotTo(BeNil())

				By("checking vip is in configmap data")
				assertVipItemInConfigMap(vipA, configmapName, true)
			})

			It("will update the configmap if another vip is added", func() {
				By("checking another vip created after trench is in configmap")
				Expect(fw.CreateResource(vipB.DeepCopy())).To(Succeed())
				assertVipItemInConfigMap(vipB, configmapName, true)
			})
		})

		Context("with two trenches", func() {
			newVip := vipB.DeepCopy()
			newVip.ObjectMeta.Labels["trench"] = "trench-b"

			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(
					&meridiov1alpha1.Trench{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "trench-b",
							Namespace: namespace,
						},
						Spec: meridiov1alpha1.TrenchSpec{
							IPFamily: string(meridiov1alpha1.IPv4),
						},
					})).To(Succeed())
			})

			It("go to its own configmap", func() {
				By("create vip for trench-a")
				Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())
				assertVipItemInConfigMap(vipA, configmapName, true)

				By("create vip for trench-b")
				Expect(fw.CreateResource(newVip)).To(Succeed())

				By("checking vip b in configmap a")
				// vip b should not be found in the configmap of trench a
				assertVipItemInConfigMap(newVip, configmapName, false)
				By("checking vip b in configmap b")
				// vip b should be found in the configmap of trench b
				assertVipItemInConfigMap(newVip, common.CMName+"-trench-b", true)
			})
		})
	})

	Context("When deleting a vip", func() {
		vp := vipA.DeepCopy()

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(vp)).To(Succeed())
			assertVipItemInConfigMap(vp, configmapName, true)
		})
		JustBeforeEach(func() {
			Expect(fw.DeleteResource(vp)).To(Succeed())
		})

		It("will update configmap", func() {
			By("checking configmap")
			assertVipItemInConfigMap(vp, configmapName, false)
		})
	})

	Context("When updating a vip", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())
		})
		It("updates the configmap", func() {
			var vp = &meridiov1alpha1.Vip{}
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(vipA), vp)).To(Succeed())
				vp.Spec.Address = "20.0.0.0/32"
				g.Expect(fw.UpdateResource(vp)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in the configmap")
			assertVipItemInConfigMap(vp, configmapName, true)

			By("checking old item is not in the configmap")
			assertVipItemInConfigMap(vipA, configmapName, false)
		})
	})

	Context("When deleting a trench", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())
		})
		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpVips()
		})

		It("will be deleted by deleting the trench", func() {
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(trench), tr)).To(Succeed())
			Expect(fw.DeleteResource(tr)).To(Succeed())

			By("checking if vip exists")
			Eventually(func() bool {
				v := &meridiov1alpha1.Vip{}
				err := fw.GetResource(client.ObjectKeyFromObject(vipA), v)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})

		It("will be deleted by deleting itself", func() {
			v := &meridiov1alpha1.Vip{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(vipA), v)).To(Succeed())
			Expect(fw.DeleteResource(v)).To(Succeed())

			By("checking if vip exists")
			Eventually(func() bool {
				v := &meridiov1alpha1.Vip{}
				err := fw.GetResource(client.ObjectKeyFromObject(vipA), v)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())

			By("checking the gateway is deleted from configmap")
			assertVipItemInConfigMap(vipA, configmapName, false)
		})
	})

	Context("when updating the configmap directly", func() {
		vp := vipA.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(vp)).To(Succeed())
			assertVipItemInConfigMap(vp, configmapName, true)
		})
		It("will be reverted according to the current vip", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vp.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking vip item still in the configmap")
			assertVipItemInConfigMap(vp, configmapName, true)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vp.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[config.VipsConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking vip item still in the configmap")
			assertVipItemInConfigMap(vp, configmapName, true)
		})
	})

	Context("checking meridio pods", func() {
		conduit := conduit(namespace)
		attractor := attractor(namespace)

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpAttractors()
			fw.CleanUpConduits()
		})

		It("will not trigger restarts in any of the meridio pods", func() {
			Expect(fw.CreateResource(vipA.DeepCopy())).To(Succeed())

			By("Checking the restarts of meridio pods")
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
		})
	})
})

func assertVipItemInConfigMap(vip *meridiov1alpha1.Vip, configmapName string, in bool) {
	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	configmap := &corev1.ConfigMap{}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, vip key has an item same as vip resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: vip.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())

		vipsconfig, err := config.UnmarshalVips(configmap.Data[config.VipsConfigKey])
		g.Expect(err).To(BeNil())

		vipmap := utils.MakeMapFromVipList(vipsconfig)
		vipInConfig, ok := vipmap[vip.ObjectMeta.Name]

		// then checking in configmap data, vip key has an item same as vip resource
		equal := equality.Semantic.DeepEqual(vipInConfig, config.Vip{
			Name:    vip.ObjectMeta.Name,
			Address: vip.Spec.Address,
			Trench:  vip.ObjectMeta.Labels["trench"]})
		return ok && equal
	}, timeout).Should(matcher)
}
