package e2e

import (
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
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

var _ = Describe("Flow", func() {
	trench := trench(namespace)

	flowA := &meridiov1alpha1.Flow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flow-a",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
			},
		},
		Spec: meridiov1alpha1.FlowSpec{
			Stream:           "stream-a",
			Protocols:        []string{"tcp"},
			SourceSubnets:    []string{"10.0.0.0/28"},
			SourcePorts:      []string{"3000"},
			DestinationPorts: []string{"2000"},
			Vips:             []string{"vip1"},
			Priority:         1,
		},
	}

	defaultFlowinCm := reader.Flow{
		Name:                  flowA.ObjectMeta.Name,
		SourceSubnets:         flowA.Spec.SourceSubnets,
		SourcePortRanges:      flowA.Spec.SourcePorts,
		DestinationPortRanges: flowA.Spec.DestinationPorts,
		Protocols:             flowA.Spec.Protocols,
		Vips:                  flowA.Spec.Vips,
		Stream:                flowA.Spec.Stream,
		Priority:              1,
	}

	BeforeEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpFlows()
	})

	AfterEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpFlows()
	})

	Context("When creating a flow", func() {
		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpFlows()
		})
		Context("without a trench", func() {
			It("will not be created", func() {
				Expect(fw.CreateResource(flowA.DeepCopy())).ToNot(Succeed())

				By("checking the existence")
				err := fw.GetResource(client.ObjectKeyFromObject(flowA), &meridiov1alpha1.Flow{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("with trench", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpTrenches()
				fw.CleanUpFlows()
			})

			It("will be created successfully", func() {
				Expect(fw.CreateResource(flowA.DeepCopy())).To(Succeed())

				By("checking if the flow exists")
				flow := &meridiov1alpha1.Flow{}
				fw.GetResource(client.ObjectKeyFromObject(flowA), flow)
				Expect(flow).NotTo(BeNil())

				By("checking flow is in configmap data")
				assertFlowItemInConfigMap(defaultFlowinCm, configmapName, true)

				By("adding another flow")
				flowB := &meridiov1alpha1.Flow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "flow-b",
						Namespace: namespace,
						Labels: map[string]string{
							"trench": trenchName,
						},
					},
					Spec: meridiov1alpha1.FlowSpec{
						Stream:           "stream-b",
						Protocols:        []string{"tcp"},
						SourceSubnets:    []string{"10.0.0.0/28"},
						SourcePorts:      []string{"any"},
						DestinationPorts: []string{"2000"},
						Vips:             []string{"vip1"},
						Priority:         1000,
					},
				}

				Expect(fw.CreateResource(flowB.DeepCopy())).To(Succeed())

				By("checking if the flow exists")
				flow = &meridiov1alpha1.Flow{}
				err := fw.GetResource(client.ObjectKeyFromObject(flowB), flow)
				Expect(err).To(BeNil())
				Expect(flow).NotTo(BeNil())

				By("checking flow is in configmap data")
				newFlowInCm := reader.Flow{
					Stream:                flowB.Spec.Stream,
					Name:                  flowB.ObjectMeta.Name,
					Protocols:             flowB.Spec.Protocols,
					SourcePortRanges:      []string{"0-65535"},
					SourceSubnets:         flowB.Spec.SourceSubnets,
					DestinationPortRanges: flowB.Spec.DestinationPorts,
					Vips:                  flowB.Spec.Vips,
					Priority:              float64(flowB.Spec.Priority),
				}
				assertFlowItemInConfigMap(newFlowInCm, configmapName, true)
			})
		})
	})

	Context("When updating a stream", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(flowA.DeepCopy())).To(Succeed())
		})
		It("updates the configmap", func() {
			var s = &meridiov1alpha1.Flow{}
			Eventually(func(g Gomega) {
				Expect(fw.GetResource(client.ObjectKeyFromObject(flowA), s)).To(Succeed())
				s.Spec.Stream = "stream-2"
				s.Spec.DestinationPorts = []string{"40000"}
				s.Spec.SourcePorts = []string{"50000"}
				s.Spec.SourceSubnets = []string{"1000::/128"}
				s.Spec.Protocols = []string{"udp"}
				s.Spec.Priority = 10
				g.Expect(fw.UpdateResource(s)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in the configmap")
			newFlowInCm := reader.Flow{
				Name:                  s.ObjectMeta.Name,
				SourceSubnets:         s.Spec.SourceSubnets,
				SourcePortRanges:      s.Spec.SourcePorts,
				DestinationPortRanges: s.Spec.DestinationPorts,
				Protocols:             s.Spec.Protocols,
				Vips:                  s.Spec.Vips,
				Priority:              float64(s.Spec.Priority),
				Stream:                s.Spec.Stream,
			}
			assertFlowItemInConfigMap(newFlowInCm, configmapName, true)

			By("checking old item is not in the configmap")
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, false)
		})

		It("will be deleted from the configmap if stream is empty", func() {
			var f = &meridiov1alpha1.Flow{}
			Eventually(func(g Gomega) {
				Expect(fw.GetResource(client.ObjectKeyFromObject(flowA), f)).To(Succeed())
				f.Spec.Stream = ""
				g.Expect(fw.UpdateResource(f)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is not in the configmap")
			fInCm := reader.Flow{
				Name:                  f.ObjectMeta.Name,
				SourceSubnets:         f.Spec.SourceSubnets,
				SourcePortRanges:      f.Spec.SourcePorts,
				DestinationPortRanges: f.Spec.DestinationPorts,
				Protocols:             f.Spec.Protocols,
				Vips:                  f.Spec.Vips,
				Priority:              float64(f.Spec.Priority),
				Stream:                f.Spec.Stream,
			}
			assertFlowItemInConfigMap(fInCm, configmapName, false)
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, false)

			By("adding the conduit back, this item will be added again")
			Eventually(func(g Gomega) {
				Expect(fw.GetResource(client.ObjectKeyFromObject(flowA), f)).To(Succeed())
				f.Spec.Stream = "stream"
				g.Expect(fw.UpdateResource(f)).To(Succeed())
			}).Should(Succeed())

			fInCm.Stream = "stream"
			By("checking new item is in the configmap")
			assertFlowItemInConfigMap(fInCm, configmapName, true)
		})
	})

	Context("When deleting", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(flowA.DeepCopy())).To(Succeed())
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, true)
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpFlows()
		})

		It("will be deleted by deleting the trench", func() {
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(trench), tr)).To(Succeed())
			Expect(fw.DeleteResource(tr)).To(Succeed())

			By("checking if flow exists")
			Eventually(func() bool {
				s := &meridiov1alpha1.Flow{}
				err := fw.GetResource(client.ObjectKeyFromObject(flowA), s)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})

		It("will be deleted by deleting itself", func() {
			s := &meridiov1alpha1.Flow{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(flowA), s)).To(Succeed())
			Expect(fw.DeleteResource(s)).To(Succeed())

			By("checking if flow exists")
			Eventually(func() bool {
				s := &meridiov1alpha1.Flow{}
				err := fw.GetResource(client.ObjectKeyFromObject(flowA), s)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())

			By("checking the flow is deleted from configmap")
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, false)
		})
	})

	Context("when updating the configmap directly", func() {
		flow := flowA.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(flow)).To(Succeed())
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, true)
		})
		It("will be reverted according to the current flow", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: flow.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking flow item still in the configmap")
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, true)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: flow.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[reader.FlowsConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking flow item still in the configmap")
			assertFlowItemInConfigMap(defaultFlowinCm, configmapName, true)
		})
	})

	Context("checking meridio pods", func() {
		attractor := attractor(namespace)
		conduit := conduit(namespace)

		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(attractor.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(conduit.DeepCopy())).To(Succeed())
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
		})
		It("will not trigger restarts in any of the meridio pods", func() {
			Expect(fw.CreateResource(flowA.DeepCopy())).To(Succeed())

			By("Checking the restarts of meridio pods")
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
		})
	})
})

func assertFlowItemInConfigMap(flow reader.Flow, configmapName string, in bool) {
	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	configmap := &corev1.ConfigMap{}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, flow key has an item same as flow resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())

		flowsconfig, err := reader.UnmarshalFlows(configmap.Data[reader.FlowsConfigKey])
		g.Expect(err).To(BeNil())

		flowmap := utils.MakeMapFromFlowList(flowsconfig)
		flowInConfig, ok := flowmap[flow.Name]

		// then checking in configmap data, flow key has an item same as flow resource
		equal := equality.Semantic.DeepEqual(flowInConfig, flow)
		return ok && equal
	}, timeout).Should(matcher)
}
