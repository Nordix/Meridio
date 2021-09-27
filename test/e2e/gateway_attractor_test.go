package e2e

import (
	"fmt"
	"time"

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

func uint32pointer(i int) *uint32 {
	ret := new(uint32)
	*ret = uint32(i)
	return ret
}

func uint16pointer(i int) *uint16 {
	ret := new(uint16)
	*ret = uint16(i)
	return ret
}

var _ = Describe("Gateway", func() {
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
	gateway := &meridiov1alpha1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gateway-a",
			Namespace: namespace,
			Labels: map[string]string{
				"attractor": attractorName,
			},
		},
		Spec: meridiov1alpha1.GatewaySpec{
			Address:  "1.2.3.4",
			Protocol: "bgp",
			Bgp: meridiov1alpha1.BgpSpec{
				RemoteASN:  uint32pointer(1234),
				LocalASN:   uint32pointer(4321),
				HoldTime:   "30s",
				RemotePort: uint16pointer(10179),
				LocalPort:  uint16pointer(10179),
			},
		},
	}
	configmapName := fmt.Sprintf("%s-%s", common.CMName, trench.ObjectMeta.Name)

	BeforeEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpGateways()
	})

	AfterEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpGateways()
	})

	Context("When creating a gateway", func() {
		AfterEach(func() {
			fw.CleanUpGateways()
		})
		Context("without trench & attractor", func() {
			It("will be created with disengaged status", func() {
				Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())

				By("checking the existence")
				gw := &meridiov1alpha1.Gateway{}
				Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				Expect(gw).NotTo(BeNil())

				By("checking the status to be disengaged")
				assertGatewayStatus(gateway, meridiov1alpha1.Disengaged)
			})
		})

		Context("with trench", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			})
			JustBeforeEach(func() {
				Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpTrenches()
			})

			Context("with attractor expecting to use this gateway", func() {
				BeforeEach(func() {
					Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
				})

				AfterEach(func() {
					fw.CleanUpAttractors()
					fw.CleanUpGateways()
				})

				It("will be created with engaged status", func() {
					By("checking if the gateway exists")
					gw := &meridiov1alpha1.Gateway{}
					Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
					Expect(gw).NotTo(BeNil())

					By("checking the status to be engaged")
					assertGatewayStatus(gateway, meridiov1alpha1.Engaged)
				})

				It("will be collected into the configmap", func() {
					By("checking gateway is in configmap data")
					gatewayItemInConfigMap(gateway, configmapName)
				})

				It("will update the gateway-in-use status in attractor", func() {
					attractorGatewayInUseEqualTo(attractor, gateway.ObjectMeta.Name)
				})

				It("will remove gateway if attractor removes it from the expected gateway list", func() {
					By("updating the attractor")
					Eventually(func(g Gomega) {
						attr := &meridiov1alpha1.Attractor{}
						g.Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr))
						attr.Spec.Gateways = []string{}
						g.Expect(fw.UpdateResource(attr)).To(Succeed())
					}).Should(Succeed())

					By("checking configmap doesn't have such item")
					gatewayItemNotInConfigMap(gateway, configmapName)

					By("checking gateway-in-use status in attractor")
					attractorGatewayInUseEqualTo(attractor)
				})
			})

			Context("with attractor not expecting to use this gateway", func() {
				BeforeEach(func() {
					newAttr := attractor.DeepCopy()
					newAttr.Spec.Gateways = []string{}
					Expect(fw.CreateResource(newAttr)).To(Succeed())
				})

				AfterEach(func() {
					fw.CleanUpAttractors()
				})

				It("will not be collected into the configmap", func() {
					By("checking configmap doesn't have such item")
					gatewayItemNotInConfigMap(gateway, configmapName)
				})

				It("will be collected if the gateway in attractor is updated", func() {
					By("updating the attractor")
					Eventually(func(g Gomega) {
						attr := &meridiov1alpha1.Attractor{}
						g.Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr))
						attr.Spec.Gateways = []string{gateway.ObjectMeta.Name}
						g.Expect(fw.UpdateResource(attr)).To(Succeed())
					}).Should(Succeed())

					By("checking gateway is in configmap")
					gatewayItemInConfigMap(gateway, configmapName)
				})
			})
		})
	})

	Context("When deleting a gateway", func() {
		gw := gateway.DeepCopy()

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(gw)).To(Succeed())
			gatewayItemInConfigMap(gw, configmapName)
		})
		JustBeforeEach(func() {
			Expect(fw.DeleteResource(gw)).To(Succeed())
		})
		It("will update configmap and attractor", func() {
			By("checking configmap")
			gatewayItemNotInConfigMap(gw, configmapName)

			By("checking gateway-in-use status in attractor")
			attractorGatewayInUseEqualTo(attractor)
		})
	})

	Context("When updating a gateway", func() {
		Context("with attractor expecting to use it", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
				Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
			})
			It("updates the configmap and attractor", func() {
				var gw = &meridiov1alpha1.Gateway{}
				Eventually(func(g Gomega) {
					g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
					gw.Spec.Address = "20.0.0.0"
					g.Expect(fw.UpdateResource(gw)).To(Succeed())
				}).Should(Succeed())

				By("checking configmap")
				gatewayItemInConfigMap(gw, configmapName)

				By("checking attractor")
				attractorGatewayInUseEqualTo(attractor, gw.ObjectMeta.Name)
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
			Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())

			By("Checking the restarts of meridio pods")
			fw.AssertTrenchReady(trench)
			fw.AssertAttractorReady(trench, attractor)
		})
	})

	Context("When deleting", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			assertAttractorStatus(attractor, meridiov1alpha1.Engaged)
			Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
			assertGatewayStatus(gateway, meridiov1alpha1.Engaged)
		})

		It("will be deleted by deleting the trench", func() {
			Expect(fw.DeleteResource(trench)).To(Succeed())
			By("checking if gateway exists")
			Eventually(func() bool {
				gw := &meridiov1alpha1.Gateway{}
				err := fw.GetResource(client.ObjectKeyFromObject(gateway), gw)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})

		It("will be deleted by deleting the attractor", func() {
			Expect(fw.DeleteResource(attractor)).To(Succeed())
			By("checking if gateway exists")
			Eventually(func() bool {
				gw := &meridiov1alpha1.Gateway{}
				err := fw.GetResource(client.ObjectKeyFromObject(gateway), gw)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})
	})

	Context("when updating the configmap directly", func() {
		gw := gateway.DeepCopy()
		attr := attractor.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(attr)).To(Succeed())
			assertAttractorStatus(attractor, meridiov1alpha1.Engaged)
			Expect(fw.CreateResource(gw)).To(Succeed())
			assertGatewayStatus(gateway, meridiov1alpha1.Engaged)
			gatewayItemInConfigMap(gw, configmapName)
		})
		It("will be reverted according to the current vip", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gw.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking gateway item still in the configmap")
			gatewayItemInConfigMap(gw, configmapName)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gw.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[config.GatewayConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking gateway item still in the configmap")
			gatewayItemInConfigMap(gw, configmapName)
		})
	})
})

func gatewayItemNotInConfigMap(gateway *meridiov1alpha1.Gateway, configmapName string) {
	configmap := &corev1.ConfigMap{}
	var ok bool
	Eventually(func(g Gomega) {
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gateway.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())
		gatewaysconfig, err := config.UnmarshalGatewayConfig(configmap.Data[config.GatewayConfigKey])
		g.Expect(err).To(BeNil())

		gatewaymap := config.MakeMapFromGWList(gatewaysconfig)
		_, ok = gatewaymap[gateway.ObjectMeta.Name]
		g.Expect(ok).To(BeFalse())
	}, timeout).Should(Succeed())
}

func gatewayItemInConfigMap(gateway *meridiov1alpha1.Gateway, configmapName string) {
	configmap := &corev1.ConfigMap{}
	var gatewayInConfig config.Gateway
	var ok bool
	Eventually(func(g Gomega) {
		// checking in configmap data, gateway key has an item same as gateway resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gateway.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())
		gatewaysconfig, err := config.UnmarshalGatewayConfig(configmap.Data[config.GatewayConfigKey])
		g.Expect(err).To(BeNil())

		gatewaymap := config.MakeMapFromGWList(gatewaysconfig)
		gatewayInConfig, ok = gatewaymap[gateway.ObjectMeta.Name]
		g.Expect(ok).To(BeTrue())

		// checking in configmap data, gateway key has an item same as gateway resource
		t, _ := time.ParseDuration(gateway.Spec.Bgp.HoldTime)
		ts := t.Seconds()
		g.Expect(gatewayInConfig).To(Equal(config.Gateway{
			Name:       gateway.ObjectMeta.Name,
			Address:    gateway.Spec.Address,
			Protocol:   "bgp",
			RemoteASN:  *gateway.Spec.Bgp.RemoteASN,
			LocalASN:   *gateway.Spec.Bgp.LocalASN,
			RemotePort: *gateway.Spec.Bgp.RemotePort,
			LocalPort:  *gateway.Spec.Bgp.LocalPort,
			IPFamily:   "ipv4",
			BFD:        false,
			HoldTime:   uint(ts),
		}))
	}, timeout).Should(Succeed())

}

func attractorGatewayInUseEqualTo(attractor *meridiov1alpha1.Attractor, gwNames ...string) {
	Eventually(func(g Gomega) {
		attr := &meridiov1alpha1.Attractor{}
		g.Expect(fw.GetResource(client.ObjectKeyFromObject(attractor), attr)).To(Succeed())
		g.Expect(attr.Status.GatewayInUse).To(Equal(gwNames))
	}, timeout).Should(Succeed())
}

func assertGatewayStatus(gateway *meridiov1alpha1.Gateway, status meridiov1alpha1.ConfigStatus) {
	Eventually(func(g Gomega) {
		gw := &meridiov1alpha1.Gateway{}
		g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
		g.Expect(gw.Status.Status).To(Equal(status))
	}).Should(Succeed())
}
