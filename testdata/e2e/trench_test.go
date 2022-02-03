package e2e

import (
	"fmt"
	"strings"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Trench", func() {
	Context("When single trench is deployed", func() {
		trench := &meridiov1alpha1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trenchName,
				Namespace: namespace,
			},
			Spec: meridiov1alpha1.TrenchSpec{
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
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKey{Name: trenchName, Namespace: namespace}, tr)).To(Succeed())
			Expect(tr.Spec.IPFamily).To(Equal(string(meridiov1alpha1.Dualstack)))
		})

		It("should fail updating the IP", func() {
			tr := &meridiov1alpha1.Trench{}
			Expect(fw.GetResource(client.ObjectKey{Name: trenchName, Namespace: namespace}, tr)).To(Succeed())
			tr.Spec.IPFamily = string(meridiov1alpha1.IPv4)
			Expect(fw.UpdateResource(tr)).ShouldNot(Succeed())
		})
	})

	Context("three trenches", func() {
		trenchA := &meridiov1alpha1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trenchName,
				Namespace: namespace,
			},
			Spec: meridiov1alpha1.TrenchSpec{
				IPFamily: string(meridiov1alpha1.Dualstack),
			},
		}

		trenchB := &meridiov1alpha1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "trench-b",
				Namespace: namespace,
			},
			Spec: meridiov1alpha1.TrenchSpec{
				IPFamily: string(meridiov1alpha1.IPv4),
			},
		}

		trenchC := &meridiov1alpha1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "trench-c",
				Namespace: namespace,
			},
			Spec: meridiov1alpha1.TrenchSpec{
				IPFamily: string(meridiov1alpha1.IPv6),
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

			By("checking role")
			roleName := fmt.Sprintf("%s-%s", common.RlName, name)
			role := &rbacv1.Role{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: name,
				Name:      roleName,
			}, role)
			Expect(apierrors.IsNotFound(err)).To(Equal(true))

			By("checking role binding")
			roleBindingName := fmt.Sprintf("%s-%s", common.RBName, name)
			roleBinding := &rbacv1.RoleBinding{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: ns,
				Name:      roleBindingName,
			}, roleBinding)
			Expect(apierrors.IsNotFound(err)).To(Equal(true))

			By("checking service account")
			serviceAccountName := fmt.Sprintf("%s-%s", common.SAName, name)
			serviceAccount := &corev1.ServiceAccount{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: ns,
				Name:      serviceAccountName,
			}, serviceAccount)
			Expect(apierrors.IsNotFound(err)).To(Equal(true))
		})
	})

})
