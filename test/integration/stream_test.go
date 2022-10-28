package e2e

import (
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

var _ = Describe("Stream", func() {
	trench := trench(namespace)
	streamA := &meridiov1alpha1.Stream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stream-a",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchName,
			},
		},
		Spec: meridiov1alpha1.StreamSpec{
			Conduit: "conduit-a",
		},
	}

	BeforeEach(func() {
		fw.CleanUpTrenches()
		fw.CleanUpAttractors()
		fw.CleanUpStreams()
		// wait for the old instances to be deleted
		time.Sleep(time.Second)
	})

	Context("When creating a stream", func() {
		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpStreams()
		})
		Context("without a trench", func() {
			It("will fail in creation", func() {
				Expect(fw.CreateResource(streamA.DeepCopy())).ToNot(Succeed())

				By("checking the existence")
				err := fw.GetResource(client.ObjectKeyFromObject(streamA), &meridiov1alpha1.Stream{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("with one trench", func() {
			BeforeEach(func() {
				Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			})
			JustBeforeEach(func() {
				Expect(fw.CreateResource(streamA.DeepCopy())).To(Succeed())
			})

			AfterEach(func() {
				fw.CleanUpTrenches()
				fw.CleanUpStreams()
			})

			It("will be created successfully", func() {
				By("checking if the stream exists")
				stream := &meridiov1alpha1.Stream{}
				err := fw.GetResource(client.ObjectKeyFromObject(streamA), stream)
				Expect(err).To(BeNil())
				Expect(stream).NotTo(BeNil())

				By("checking stream is in configmap data")
				assertStreamItemInConfigMap(streamA, configmapName, true)
			})
		})

		Context("with two trenches", func() {
			streamB := streamA.DeepCopy()
			streamB.ObjectMeta.Name = "stream-b"
			streamB.ObjectMeta.Labels["trench"] = "trench-b"
			streamB.Spec.Conduit = "conduit-b"

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
				By("create stream for trench-a")
				Expect(fw.CreateResource(streamA.DeepCopy())).To(Succeed())
				assertStreamItemInConfigMap(streamA, configmapName, true)

				By("create stream for trench-b")
				Expect(fw.CreateResource(streamB)).To(Succeed())

				By("checking stream b in configmap a")
				// stream b should not be found in the configmap of trench a
				assertStreamItemInConfigMap(streamB, configmapName, false)
				By("checking stream b in configmap b")
				// stream b should be found in the configmap of trench b
				assertStreamItemInConfigMap(streamB, common.CMName+"-trench-b", true)
			})
		})
	})

	Context("When updating a stream", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(streamA.DeepCopy())).To(Succeed())
		})
		It("updates the configmap", func() {
			var s = &meridiov1alpha1.Stream{}
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(streamA), s)).To(Succeed())
				s.Spec.Conduit = "conduit-2"
				g.Expect(fw.UpdateResource(s)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in the configmap")
			assertStreamItemInConfigMap(s, configmapName, true)

			By("checking old item is not in the configmap")
			assertStreamItemInConfigMap(streamA, configmapName, false)
		})

		It("will be deleted from the configmap if conduit is empty", func() {
			var s = &meridiov1alpha1.Stream{}
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(streamA), s)).To(Succeed())
				s.Spec.Conduit = ""
				g.Expect(fw.UpdateResource(s)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is not in the configmap")
			assertStreamItemInConfigMap(s, configmapName, false)
			assertStreamItemInConfigMap(streamA, configmapName, false)

			By("adding the conduit back, this item will be added again")
			Eventually(func(g Gomega) {
				g.Expect(fw.GetResource(client.ObjectKeyFromObject(s), s)).To(Succeed())
				s.Spec.Conduit = "conduit"
				g.Expect(fw.UpdateResource(s)).To(Succeed())
			}).Should(Succeed())

			By("checking new item is in the configmap")
			assertStreamItemInConfigMap(s, configmapName, true)
		})
	})

	Context("When deleting", func() {
		BeforeEach(func() {
			Expect(fw.CreateResource(trench.DeepCopy())).To(Succeed())
			Expect(fw.CreateResource(streamA.DeepCopy())).To(Succeed())
			assertStreamItemInConfigMap(streamA, configmapName, true)
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
			fw.CleanUpStreams()
		})

		It("will be deleted by deleting the trench", func() {
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(trench), tr)).To(Succeed())
			Expect(fw.DeleteResource(tr)).To(Succeed())

			By("checking if stream exists")
			Eventually(func() bool {
				s := &meridiov1alpha1.Stream{}
				err := fw.GetResource(client.ObjectKeyFromObject(streamA), s)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())
		})

		It("will be deleted by deleting itself", func() {
			s := &meridiov1alpha1.Stream{}
			Expect(fw.GetResource(client.ObjectKeyFromObject(streamA), s)).To(Succeed())
			Expect(fw.DeleteResource(s)).To(Succeed())

			By("checking if stream exists")
			Eventually(func() bool {
				s := &meridiov1alpha1.Stream{}
				err := fw.GetResource(client.ObjectKeyFromObject(streamA), s)
				return err != nil && apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue())

			By("checking the stream is deleted from configmap")
			assertStreamItemInConfigMap(streamA, configmapName, false)
		})
	})

	Context("when updating the configmap directly", func() {
		stream := streamA.DeepCopy()
		tr := trench.DeepCopy()
		BeforeEach(func() {
			Expect(fw.CreateResource(tr)).To(Succeed())
			Expect(fw.CreateResource(stream)).To(Succeed())
			assertStreamItemInConfigMap(stream, configmapName, true)
		})
		It("will be reverted according to the current stream", func() {
			By("deleting the configmap")
			configmap := &corev1.ConfigMap{}
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: stream.ObjectMeta.Namespace}, configmap)).To(Succeed())
			Expect(fw.DeleteResource(configmap)).To(Succeed())

			By("checking stream item still in the configmap")
			assertStreamItemInConfigMap(stream, configmapName, true)

			By("updating the configmap")
			Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: stream.ObjectMeta.Namespace}, configmap)).To(Succeed())
			configmap.Data[config.StreamsConfigKey] = ""
			Eventually(func(g Gomega) {
				g.Expect(fw.UpdateResource(configmap)).To(Succeed())
			}).Should(Succeed())

			By("checking stream item still in the configmap")
			assertStreamItemInConfigMap(stream, configmapName, true)
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
			Expect(fw.CreateResource(streamA.DeepCopy())).To(Succeed())

			By("Checking the restarts of meridio pods")
			AssertMeridioDeploymentsReady(trench, attractor, conduit)
		})
	})
})

func assertStreamItemInConfigMap(stream *meridiov1alpha1.Stream, configmapName string, in bool) {
	matcher := BeFalse()
	if in {
		matcher = BeTrue()
	}
	configmap := &corev1.ConfigMap{}
	Eventually(func(g Gomega) bool {
		// checking in configmap data, stream key has an item same as stream resource
		g.Expect(fw.GetResource(client.ObjectKey{Name: configmapName, Namespace: stream.ObjectMeta.Namespace}, configmap)).To(Succeed())
		g.Expect(configmap).ToNot(BeNil())

		streamsconfig, err := config.UnmarshalStreams(configmap.Data[config.StreamsConfigKey])
		g.Expect(err).To(BeNil())

		streammap := utils.MakeMapFromStreamList(streamsconfig)
		streamInConfig, ok := streammap[stream.ObjectMeta.Name]

		// then checking in configmap data, stream key has an item same as stream resource
		equal := equality.Semantic.DeepEqual(streamInConfig, config.Stream{
			Name:    stream.ObjectMeta.Name,
			Conduit: stream.Spec.Conduit})
		return ok && equal
	}, timeout).Should(matcher)
}
