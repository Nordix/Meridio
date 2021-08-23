package e2e_test

import (
	"context"
	"fmt"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
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

		It("should have the trench pods in running state", func() {
			By("checking if proxy pods are in running state")
			proxyName := fmt.Sprintf("proxy-%s", trench.ObjectMeta.Name)
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
			ipamName := fmt.Sprintf("ipam-%s", trench.ObjectMeta.Name)
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
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), ipamName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(int(deployment.Status.Replicas)))
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}

			By("checking if nsp pods are in running state")
			nspName := fmt.Sprintf("nsp-%s", trench.ObjectMeta.Name)
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
			deployment, err = clientset.AppsV1().Deployments(namespace).Get(context.Background(), nspName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(int(deployment.Status.Replicas)))
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}
		})
	})

})
