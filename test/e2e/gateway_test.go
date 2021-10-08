package e2e

import (
	"fmt"
	"net"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"github.com/nordix/meridio-operator/test/utils"
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
	trench := trench(namespace)
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
			It("will fail in creation", func() {
				Expect(fw.CreateResource(gateway.DeepCopy())).ToNot(Succeed())

				By("checking the existence")
				err := fw.GetResource(client.ObjectKeyFromObject(gateway), &meridiov1alpha1.Gateway{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
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

			It("will be created successfully", func() {
				By("checking if the gateway exists")
				gw := &meridiov1alpha1.Gateway{}
				Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				Expect(gw).NotTo(BeNil())

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
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpGateways()
		})

		It("will be deleted by deleting the trench", func() {
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(trench), tr)).To(Succeed())
			Expect(fw.DeleteResource(tr)).To(Succeed())
			By("checking if gateway exists")
			Eventually(func() bool {
				g := &meridiov1alpha1.Gateway{}
				err := fw.GetResource(client.ObjectKeyFromObject(gateway), g)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})

		It("will be deleted by deleting itself", func() {
			gw := &meridiov1alpha1.Gateway{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
			Expect(fw.DeleteResource(gw)).To(Succeed())
			By("checking if gateway exists")
			Eventually(func() bool {
				g := &meridiov1alpha1.Gateway{}
				err := fw.GetResource(client.ObjectKeyFromObject(gateway), g)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())

			By("checking the gateway is deleted from configmap")
			assertGatewayItemInConfigMap(gateway, configmapName, false)
		})
	})

	Context("when updating the configmap directly", func() {
		gw := gateway.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(gw)).To(Succeed())
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
		conduit := conduit(namespace)
		attractor := attractor(namespace)

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
		})
		It("will not trigger restarts in any of the meridio pods", func() {
			Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())

			By("Checking the restarts of meridio pods")
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
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

		gatewaymap := utils.MakeMapFromGWList(gatewaysconfig)
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
