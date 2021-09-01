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

var _ = Describe("Attractor", func() {

	Context("When an attractor is deployed", func() {

		var (
			trench    *meridiov1alpha1.Trench
			attractor *meridiov1alpha1.Attractor
			replicas  *int32
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
			replicas = new(int32)
			*replicas = 1
			attractor = &meridiov1alpha1.Attractor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      attractorName,
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trenchName,
					},
				},
				Spec: meridiov1alpha1.AttractorSpec{
					VlanID:        100,
					VlanInterface: "eth0",
					Replicas:      replicas,
					Gateways:      []string{},
					Vips:          []string{},
				},
			}
			Expect(kubeAPIClient.Create(context.Background(), attractor)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(kubeAPIClient.Delete(context.Background(), attractor)).Should(Succeed())
			Expect(kubeAPIClient.Delete(context.Background(), trench)).Should(Succeed())
		})

		It("should have the attractor pods in running state", func() {
			By("checking if lb-fe pods are in running state")
			loadBalancerName := fmt.Sprintf("%s-%s", common.LBName, trench.ObjectMeta.Name)
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), loadBalancerName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
			listOptions := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", loadBalancerName),
			}
			pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), loadBalancerName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}

			By("checking if nse-vlan pods are in running state")
			nseVLANName := fmt.Sprintf("%s-%s", common.NseName, attractorName)
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), nseVLANName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
			listOptions = metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", nseVLANName),
			}
			pods, err = clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			deployment, err = clientset.AppsV1().Deployments(namespace).Get(context.Background(), nseVLANName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(int(deployment.Status.Replicas)).To(Equal(int(*replicas)))
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}

			By("checking if configmap has been created")
			configmapName := fmt.Sprintf("%s-%s", common.CMName, trench.ObjectMeta.Name)
			configmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(configmap).ToNot(BeNil())
		})
	})
})
