package integration_test

import (
	"fmt"
	"strings"
	"time"

	meridiov1 "github.com/nordix/meridio/api/v1"
	"github.com/nordix/meridio/pkg/controllers/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Trench", func() {
	Context("When single trench is deployed", func() {
		trench := &meridiov1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trenchName,
				Namespace: namespace,
			},
			Spec: meridiov1.TrenchSpec{
				IPFamily: "dualstack",
			},
		}

		BeforeEach(func() {
			fw.CleanUpTrenches()
			// wait for the old instances to be deleted
			time.Sleep(time.Second)
			Expect(fw.CreateResource(trench.DeepCopy())).Should(Succeed())
		})

		It("should have the trench pods in running state and the resources created", func() {
			AssertTrenchReady(trench)
		})

		It("has default IP family to be dual stack", func() {
			tr := &meridiov1.Trench{}
			Expect(fw.GetResource(client.ObjectKey{Name: trenchName, Namespace: namespace}, tr)).To(Succeed())
			Expect(tr.Spec.IPFamily).To(Equal(string(meridiov1.Dualstack)))
		})

		It("should fail updating the IP", func() {
			tr := &meridiov1.Trench{}
			Expect(fw.GetResource(client.ObjectKey{Name: trenchName, Namespace: namespace}, tr)).To(Succeed())
			tr.Spec.IPFamily = string(meridiov1.IPv4)
			Expect(fw.UpdateResource(tr)).ShouldNot(Succeed())
		})
	})

	Context("three trenches", func() {
		trenchA := &meridiov1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trenchName,
				Namespace: namespace,
			},
			Spec: meridiov1.TrenchSpec{
				IPFamily: string(meridiov1.Dualstack),
			},
		}

		trenchB := &meridiov1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "trench-b",
				Namespace: namespace,
			},
			Spec: meridiov1.TrenchSpec{
				IPFamily: string(meridiov1.IPv4),
			},
		}

		trenchC := &meridiov1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "trench-c",
				Namespace: namespace,
			},
			Spec: meridiov1.TrenchSpec{
				IPFamily: string(meridiov1.IPv6),
			},
		}

		BeforeEach(func() {
			fw.CleanUpTrenches()
			// wait for the old instances to be deleted
			time.Sleep(time.Second)
			Expect(fw.CreateResource(trenchA.DeepCopy())).Should(Succeed())
			Expect(fw.CreateResource(trenchB.DeepCopy())).Should(Succeed())
			Expect(fw.CreateResource(trenchC.DeepCopy())).Should(Succeed())
		})

		It("creates resources of three trenches", func() {
			AssertTrenchReady(trenchA)
			AssertTrenchReady(trenchB)
			AssertTrenchReady(trenchC)
		})

		It("will delete one trench will delete all relevant resources of the same trench", func() {
			By("deleting trench A")
			Expect(fw.DeleteResource(trenchA)).Should(Succeed())

			By("checking the other two trenches are not affected")
			Context("three trenches are ready", func() {
				AssertTrenchReady(trenchB)
				AssertTrenchReady(trenchC)
			})

			By("checking resources from trench A are not existing")
			ns := trenchA.ObjectMeta.Namespace
			name := trenchA.ObjectMeta.Name
			By("checking ipam StatefulSet")
			Eventually(func(g Gomega) {
				g.Expect(assertStatefulSetReady(strings.Join([]string{"ipam", name}, "-"), ns)).Should(Succeed())
			}, timeout, interval).ShouldNot(Succeed())

			By("checking nsp StatefulSet")
			Eventually(func(g Gomega) {
				g.Expect(assertStatefulSetReady(strings.Join([]string{"nsp", name}, "-"), ns)).Should(Succeed())
			}, timeout, interval).ShouldNot(Succeed())

			By("checking nsp service")
			nspServiceName := fmt.Sprintf("%s-%s", common.NspSvcName, name)
			service := &corev1.Service{}
			err := fw.GetResource(client.ObjectKey{
				Namespace: ns,
				Name:      nspServiceName,
			}, service)
			Expect(apierrors.IsNotFound(err)).To(Equal(true))

			By("checking ipam service")
			ipamServiceName := fmt.Sprintf("%s-%s", common.IpamSvcName, name)
			ipamService := &corev1.Service{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: ns,
				Name:      ipamServiceName,
			}, ipamService)
			Expect(apierrors.IsNotFound(err)).To(Equal(true))
		})
	})

})
