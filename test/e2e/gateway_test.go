package e2e

import (
	"fmt"
	"net"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	configutils "github.com/nordix/meridio-operator/controllers/config"
	"github.com/nordix/meridio/pkg/configuration/reader"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
	gateway := &meridiov1alpha1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gateway-a",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
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
		fw.CleanUpGateways()
	})

	AfterEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpGateways()
	})

	Context("When creating a gateway", func() {
		AfterEach(func() {
			fw.CleanUpGateways()
		})
		Context("without trench", func() {
			It("will be created with disengaged status", func() {
				Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())

				By("checking the existence")
				gw := &meridiov1alpha1.Gateway{}
				Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				Expect(gw).NotTo(BeNil())

				By("checking the status to be disengaged")
				assertGatewayStatus(gateway, meridiov1alpha1.Disengaged)

				By("checking this gateway is not in the configmap")
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
				assertGatewayItemInConfigMap(gateway, configmapName, false)

				gatewayB := &meridiov1alpha1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gateway-b",
						Namespace: namespace,
						Labels: map[string]string{
							"trench": trenchName,
						},
					},
					Spec: meridiov1alpha1.GatewaySpec{
						Address:  "1000::",
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

				By("checking another gateway created after trench is in configmap")
				Expect(fw.CreateResource(gatewayB.DeepCopy())).To(Succeed())
				assertGatewayItemInConfigMap(gatewayB, configmapName, true)
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
				assertGatewayItemInConfigMap(gateway, configmapName, true)
			})
		})
	})

	Context("When deleting a gateway", func() {
		gw := gateway.DeepCopy()

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(gw)).To(Succeed())
			assertGatewayItemInConfigMap(gw, configmapName, true)
		})
		JustBeforeEach(func() {
			Expect(fw.DeleteResource(gw)).To(Succeed())
		})
		It("will update configmap", func() {
			By("checking the gateway is deleted from the configmap")
			assertGatewayItemInConfigMap(gw, configmapName, false)
		})
	})

	Context("When updating a gateway", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
		})
		It("updates the configmap", func() {
			var gw = &meridiov1alpha1.Gateway{}
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				gw.Spec.Address = "20.0.0.0"
				g.Expect(fw.UpdateResource(gw)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in configmap")
			assertGatewayItemInConfigMap(gw, configmapName, true)

			By("checking old item is not in configmap")
			assertGatewayItemInConfigMap(gateway, configmapName, false)
		})
	})

	Context("When deleting", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
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

		It("will be deleted by deleting itself", func() {
			Expect(fw.DeleteResource(gateway)).To(Succeed())
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
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(gw)).To(Succeed())
			assertGatewayStatus(gateway, meridiov1alpha1.Engaged)
			assertGatewayItemInConfigMap(gw, configmapName, true)
		})
		It("will be reverted according to the current vip", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gw.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking gateway item still in the configmap")
			assertGatewayItemInConfigMap(gw, configmapName, true)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gw.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[reader.GatewaysConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking gateway item still in the configmap")
			assertGatewayItemInConfigMap(gw, configmapName, true)
		})
	})

	Context("checking meridio pods", func() {
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
})

func assertGatewayItemInConfigMap(gateway *meridiov1alpha1.Gateway, configmapName string, in bool) {
	configmap := &corev1.ConfigMap{}

	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, gateway key has an item same as gateway resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gateway.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())
		gatewaysconfig, err := reader.UnmarshalGateways(configmap.Data[reader.GatewaysConfigKey])
		g.Expect(err).To(BeNil())

		gatewaymap := configutils.MakeMapFromGWList(gatewaysconfig)
		gatewayInConfig, ok := gatewaymap[gateway.ObjectMeta.Name]

		// checking in configmap data, gateway key has an item same as gateway resource
		t, _ := time.ParseDuration(gateway.Spec.Bgp.HoldTime)
		ts := t.Seconds()
		ipf := "ipv4"
		if net.ParseIP(gateway.Spec.Address).To4() == nil {
			ipf = "ipv6"
		}
		equal := equality.Semantic.DeepEqual(gatewayInConfig, reader.Gateway{
			Name:       gateway.ObjectMeta.Name,
			Address:    gateway.Spec.Address,
			Protocol:   "bgp",
			RemoteASN:  *gateway.Spec.Bgp.RemoteASN,
			LocalASN:   *gateway.Spec.Bgp.LocalASN,
			RemotePort: *gateway.Spec.Bgp.RemotePort,
			LocalPort:  *gateway.Spec.Bgp.LocalPort,
			IPFamily:   ipf,
			BFD:        false,
			HoldTime:   uint(ts),
			Trench:     gateway.ObjectMeta.Labels["trench"],
		})
		return ok && equal
	}, timeout).Should(matcher)

}

func assertGatewayStatus(gateway *meridiov1alpha1.Gateway, status meridiov1alpha1.ConfigStatus) {
	Eventually(func(g Gomega) {
		gw := &meridiov1alpha1.Gateway{}
		g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
		g.Expect(gw.Status.Status).To(Equal(status))
	}).Should(Succeed())
}
