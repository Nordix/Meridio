package e2e_test

import (
	"context"
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Trench", func() {

	Context("When a trench is deployed", func() {
		var (
			trench *meridiov1alpha1.Trench
		)

		BeforeEach(func() {
			trench = &meridiov1alpha1.Trench{
				ObjectMeta: metav1.ObjectMeta{
					Name:      trenchName,
					Namespace: namespace,
				},
				Spec: meridiov1alpha1.TrenchSpec{
					IPFamily: "DualStack",
				},
			}
			Expect(kubeAPIClient.Create(context.Background(), trench)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(kubeAPIClient.Delete(context.Background(), trench)).Should(Succeed())
		})

		It("should have the trench pods in running state and the resources created", func() {
			By("checking if proxy pods are in running state")
			proxyName := fmt.Sprintf("%s-%s", common.ProxyName, trench.ObjectMeta.Name)
			Eventually(func() bool {
				daemonset, err := clientset.AppsV1().DaemonSets(namespace).Get(context.Background(), proxyName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return daemonset.Status.DesiredNumberScheduled == daemonset.Status.NumberReady
			}, timeout, interval).Should(BeTrue())
			listOptions := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", proxyName),
			}
			pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			daemonset, err := clientset.AppsV1().DaemonSets(namespace).Get(context.Background(), proxyName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(int(daemonset.Status.DesiredNumberScheduled)))
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}

			By("checking if ipam pods are in running state")
			ipamName := fmt.Sprintf("%s-%s", common.IpamName, trench.ObjectMeta.Name)
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), ipamName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
			listOptions = metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", ipamName),
			}
			pods, err = clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}

			By("checking if nsp pods are in running state")
			nspName := fmt.Sprintf("%s-%s", common.NspName, trench.ObjectMeta.Name)
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), nspName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
			listOptions = metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", nspName),
			}
			pods, err = clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}

			By("checking if nsp service has been created")
			nspServiceName := fmt.Sprintf("%s-%s", common.NspSvcName, trench.ObjectMeta.Name)
			service, err := clientset.CoreV1().Services(namespace).Get(context.Background(), nspServiceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service).ToNot(BeNil())

			By("checking if ipam service has been created")
			ipamServiceName := fmt.Sprintf("%s-%s", common.IpamSvcName, trench.ObjectMeta.Name)
			service, err = clientset.CoreV1().Services(namespace).Get(context.Background(), ipamServiceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service).ToNot(BeNil())

			By("checking if role has been created")
			roleName := fmt.Sprintf("%s-%s", common.RlName, trench.ObjectMeta.Name)
			role, err := clientset.RbacV1().Roles(namespace).Get(context.Background(), roleName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(role).ToNot(BeNil())

			By("checking if role binding has been created")
			roleBindingName := fmt.Sprintf("%s-%s", common.RBName, trench.ObjectMeta.Name)
			roleBinding, err := clientset.RbacV1().RoleBindings(namespace).Get(context.Background(), roleBindingName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(roleBinding).ToNot(BeNil())

			By("checking if service account has been created")
			serviceAccountName := fmt.Sprintf("%s-%s", common.SAName, trench.ObjectMeta.Name)
			serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.Background(), serviceAccountName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(serviceAccount).ToNot(BeNil())
		})
	})

})
