package e2e

import (
	"fmt"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	"github.com/nordix/meridio-operator/testdata/utils"
	"github.com/nordix/meridio/pkg/configuration/reader"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
	// gateway item in configmap if default gateway is used
	defaultGwInCm := reader.Gateway{
		Name:       gateway.ObjectMeta.Name,
		Address:    "1.2.3.4",
		Protocol:   "bgp",
		RemoteASN:  1234,
		LocalASN:   4321,
		RemotePort: 10179,
		LocalPort:  10179,
		HoldTime:   30,
		BFD:        false,
		IPFamily:   "ipv4",
		Trench:     trenchName,
	}

	configmapName := fmt.Sprintf("%s-%s", common.CMName, trench.ObjectMeta.Name)

	BeforeEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpGateways()
		// wait for the old instances to be deleted
		time.Sleep(time.Second)
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

		Context("with one trench", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpTrenches()
			})

			It("will be created successfully", func() {
				Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())

				By("checking if the gateway exists")
				gw := &meridiov1alpha1.Gateway{}
				Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				Expect(gw).NotTo(BeNil())

				By("checking gateway is in configmap data")

				assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)
			})

			Context("mutating webhook", func() {
				gatewayIncomplete := &meridiov1alpha1.Gateway{
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
							RemoteASN: uint32pointer(1234),
							LocalASN:  uint32pointer(4321),
							HoldTime:  "30s",
							BFD: meridiov1alpha1.BfdSpec{
								Switch: pointer.Bool(true),
							},
						},
					},
				}

				gwInCm := reader.Gateway{
					Name:       gateway.ObjectMeta.Name,
					Address:    "1.2.3.4",
					Protocol:   "bgp",
					RemoteASN:  1234,
					LocalASN:   4321,
					RemotePort: 179,
					LocalPort:  179,
					HoldTime:   30,
					BFD:        true,
					MinTx:      300,
					MinRx:      300,
					Multiplier: 3,
					IPFamily:   "ipv4",
					Trench:     trenchName,
				}

				It("will have different result", func() {
					if mutating {
						By("if mutating webhook is enabled, gateway will be created successfully")
						Expect(fw.CreateResource(gatewayIncomplete.DeepCopy())).To(Succeed())

						By("checking if the gateway exists")
						gw := &meridiov1alpha1.Gateway{}
						Expect(fw.GetResource(client.ObjectKeyFromObject(gatewayIncomplete), gw)).To(Succeed())
						Expect(gw).NotTo(BeNil())

						By("checking gateway is in configmap data")

						assertGatewayItemInConfigMap(gwInCm, configmapName, true)
					} else {
						By("if mutating webhook is disabled, gateway will fail to be created")
						Expect(fw.CreateResource(gatewayIncomplete.DeepCopy())).ToNot(Succeed())
					}
				})
			})
		})

		Context("with two trenches", func() {
			gatewayB := gateway.DeepCopy()
			gatewayB.ObjectMeta.Name = "new-gw"
			gatewayB.ObjectMeta.Labels["trench"] = "trench-b"
			gatewayB.Spec.Address = "10.20.30.40"

			newGatewayInCm := reader.Gateway{
				Name:       gatewayB.ObjectMeta.Name,
				Address:    gatewayB.Spec.Address,
				Protocol:   "bgp",
				RemoteASN:  1234,
				LocalASN:   4321,
				RemotePort: 10179,
				LocalPort:  10179,
				HoldTime:   30,
				BFD:        false,
				IPFamily:   "ipv4",
				Trench:     "trench-b",
			}

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
				By("create gateway for trench-a")
				Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
				assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)

				By("create gateway for trench-b")
				Expect(fw.CreateResource(gatewayB)).To(Succeed())

				By("checking gateway b in configmap a")
				// gateway b should not be found in the configmap of trench a
				assertGatewayItemInConfigMap(newGatewayInCm, common.CMName+"-trench-a", false)
				By("checking gateway b in configmap b")
				// gateway b should be found in the configmap of trench b
				assertGatewayItemInConfigMap(newGatewayInCm, common.CMName+"-trench-b", true)
			})
		})
	})

	Context("When deleting a gateway", func() {
		gw := gateway.DeepCopy()

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(gw)).To(Succeed())
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)
		})
		JustBeforeEach(func() {
			Expect(fw.DeleteResource(gw)).To(Succeed())
		})
		It("will update configmap", func() {
			By("checking the gateway is deleted from the configmap")
			defaultGwInCm := reader.Gateway{
				Name: gateway.ObjectMeta.Name,
			}
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, false)
		})
	})

	Context("When updating a gateway", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)
		})

		AfterEach(func() {
			fw.CleanUpGateways()
		})

		It("updates the configmap when gateway address is updated", func() {
			var gw = &meridiov1alpha1.Gateway{}
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				gw.Spec.Address = "20.0.0.0"
				g.Expect(fw.UpdateResource(gw)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in configmap")
			newItem := defaultGwInCm
			newItem.Address = "20.0.0.0"
			assertGatewayItemInConfigMap(newItem, configmapName, true)

			By("checking old item is not in configmap")
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, false)
		})

		It("updates the configmap when gateway address is updated", func() {
			var gw = &meridiov1alpha1.Gateway{}
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				gw.Spec.Bgp.BFD = meridiov1alpha1.BfdSpec{
					Switch:     pointer.Bool(true),
					MinRx:      "300ms",
					MinTx:      "300ms",
					Multiplier: uint16pointer(3),
				}
				g.Expect(fw.UpdateResource(gw)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in configmap")
			newItem := defaultGwInCm
			newItem.BFD = true
			newItem.MinRx = 300
			newItem.MinTx = 300
			newItem.Multiplier = 3
			assertGatewayItemInConfigMap(newItem, configmapName, true)

			By("checking old item is not in configmap")
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, false)
		})

		It("when gateway protocol is updated", func() {
			var gw = &meridiov1alpha1.Gateway{}
			By("checking update fail when protocol is static but bgp section exist")
			Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
			gw.Spec.Protocol = string(meridiov1alpha1.Static)
			Expect(fw.UpdateResource(gw)).ToNot(Succeed())

			By("checking update succeed when protocol is static and bgp section is removed")
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				gw.Spec.Protocol = "static"
				gw.Spec.Static = meridiov1alpha1.StaticSpec{
					BFD: meridiov1alpha1.BfdSpec{
						Switch:     pointer.Bool(true),
						MinRx:      "200ms",
						MinTx:      "200ms",
						Multiplier: uint16pointer(5),
					},
				}
				gw.Spec.Bgp = meridiov1alpha1.BgpSpec{}
				g.Expect(fw.UpdateResource(gw)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in configmap")
			newItem := reader.Gateway{
				Name:       gw.ObjectMeta.Name,
				Address:    "1.2.3.4",
				IPFamily:   "ipv4",
				Protocol:   "static",
				BFD:        true,
				MinTx:      200,
				MinRx:      200,
				Multiplier: 5,
				Trench:     trenchName,
			}
			assertGatewayItemInConfigMap(newItem, configmapName, true)

			By("checking update succeed when protocol is static and bfd section is removed")
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(gateway), gw)).To(Succeed())
				gw.Spec.Protocol = "static"
				gw.Spec.Static = meridiov1alpha1.StaticSpec{}
				gw.Spec.Bgp = meridiov1alpha1.BgpSpec{}
				g.Expect(fw.UpdateResource(gw)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in configmap")
			// if mutating webhook is enabled, BFD is turned on for static protocol by default
			if mutating {
				newItem = reader.Gateway{
					Name:       gw.ObjectMeta.Name,
					Address:    "1.2.3.4",
					IPFamily:   "ipv4",
					Protocol:   "static",
					BFD:        true,
					MinRx:      200,
					MinTx:      200,
					Multiplier: 3,
					Trench:     trenchName,
				}
			} else {
				newItem = reader.Gateway{
					Name:     gw.ObjectMeta.Name,
					Address:  "1.2.3.4",
					IPFamily: "ipv4",
					Protocol: "static",
					Trench:   trenchName,
					BFD:      false,
				}
			}

			assertGatewayItemInConfigMap(newItem, configmapName, true)

			By("checking old item is not in configmap")
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, false)
		})
	})

	Context("When deleting", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(gateway.DeepCopy())).To(Succeed())
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)
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
			defaultGwInCm := reader.Gateway{
				Name: gateway.ObjectMeta.Name,
			}
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, false)
		})
	})

	Context("when updating the configmap directly", func() {
		gw := gateway.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(gw)).To(Succeed())
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)
		})
		It("will be reverted according to the current gateway", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gw.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking gateway item still in the configmap")
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: gw.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[reader.GatewaysConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking gateway item still in the configmap")
			assertGatewayItemInConfigMap(defaultGwInCm, configmapName, true)
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

func assertGatewayItemInConfigMap(gateway reader.Gateway, configmapName string, in bool) {
	configmap := &corev1.ConfigMap{}

	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, gateway key has an item same as gateway resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())
		gatewaysconfig, err := reader.UnmarshalGateways(configmap.Data[reader.GatewaysConfigKey])
		g.Expect(err).To(BeNil())

		gatewaymap := utils.MakeMapFromGWList(gatewaysconfig)
		gatewayInConfig, ok := gatewaymap[gateway.Name]

		equal := equality.Semantic.DeepEqual(gatewayInConfig, gateway)
		return ok && equal
	}, timeout).Should(matcher)
}
