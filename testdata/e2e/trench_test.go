package e2e

import (
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Trench", func() {

	Context("When a trench is deployed", func() {
		trench := &meridiov1alpha1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trenchName,
				Namespace: namespace,
			},
		}

		BeforeEach(func() {
			fw.CleanUpTrenches()
			Expect(fw.CreateResource(trench.DeepCopy())).Should(Succeed())
		})

		AfterEach(func() {
			fw.CleanUpTrenches()
		})

		FIt("should have the trench pods in running state and the resources created", func() {
			AssertTrenchReady(trench)

			By("checking if nsp service has been created")
			nspServiceName := fmt.Sprintf("%s-%s", common.NspSvcName, trench.ObjectMeta.Name)
			service := &corev1.Service{}
			err := fw.GetResource(client.ObjectKey{
				Namespace: trench.ObjectMeta.Namespace,
				Name:      nspServiceName,
			}, service)
			Expect(err).ToNot(HaveOccurred())
			Expect(service).ToNot(BeNil())

			By("checking if ipam service has been created")
			ipamServiceName := fmt.Sprintf("%s-%s", common.IpamSvcName, trench.ObjectMeta.Name)
			ipamService := &corev1.Service{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: trench.ObjectMeta.Namespace,
				Name:      ipamServiceName,
			}, ipamService)
			Expect(err).ToNot(HaveOccurred())
			Expect(service).ToNot(BeNil())

			By("checking if role has been created")
			roleName := fmt.Sprintf("%s-%s", common.RlName, trench.ObjectMeta.Name)
			role := &rbacv1.Role{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: trench.ObjectMeta.Namespace,
				Name:      roleName,
			}, role)
			Expect(err).ToNot(HaveOccurred())
			Expect(role).ToNot(BeNil())

			By("checking if role binding has been created")
			roleBindingName := fmt.Sprintf("%s-%s", common.RBName, trench.ObjectMeta.Name)
			roleBinding := &rbacv1.RoleBinding{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: trench.ObjectMeta.Namespace,
				Name:      roleBindingName,
			}, roleBinding)
			Expect(err).ToNot(HaveOccurred())
			Expect(roleBinding).ToNot(BeNil())

			By("checking if service account has been created")
			serviceAccountName := fmt.Sprintf("%s-%s", common.SAName, trench.ObjectMeta.Name)
			serviceAccount := &corev1.ServiceAccount{}
			err = fw.GetResource(client.ObjectKey{
				Namespace: trench.ObjectMeta.Namespace,
				Name:      serviceAccountName,
			}, serviceAccount)
			Expect(err).ToNot(HaveOccurred())
			Expect(serviceAccount).ToNot(BeNil())
		})
	})

})
