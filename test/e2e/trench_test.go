package e2e

import (
	"context"
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

		It("should have the trench pods in running state and the resources created", func() {
			fw.AssertTrenchReady(trench)

			By("checking if nsp service has been created")
			nspServiceName := fmt.Sprintf("%s-%s", common.NspSvcName, trench.ObjectMeta.Name)
			service, err := fw.Clientset.CoreV1().Services(namespace).Get(context.Background(), nspServiceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service).ToNot(BeNil())

			By("checking if ipam service has been created")
			ipamServiceName := fmt.Sprintf("%s-%s", common.IpamSvcName, trench.ObjectMeta.Name)
			service, err = fw.Clientset.CoreV1().Services(namespace).Get(context.Background(), ipamServiceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service).ToNot(BeNil())

			By("checking if role has been created")
			roleName := fmt.Sprintf("%s-%s", common.RlName, trench.ObjectMeta.Name)
			role, err := fw.Clientset.RbacV1().Roles(namespace).Get(context.Background(), roleName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(role).ToNot(BeNil())

			By("checking if role binding has been created")
			roleBindingName := fmt.Sprintf("%s-%s", common.RBName, trench.ObjectMeta.Name)
			roleBinding, err := fw.Clientset.RbacV1().RoleBindings(namespace).Get(context.Background(), roleBindingName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(roleBinding).ToNot(BeNil())

			By("checking if service account has been created")
			serviceAccountName := fmt.Sprintf("%s-%s", common.SAName, trench.ObjectMeta.Name)
			serviceAccount, err := fw.Clientset.CoreV1().ServiceAccounts(namespace).Get(context.Background(), serviceAccountName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(serviceAccount).ToNot(BeNil())
		})
	})

})
