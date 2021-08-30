package e2e_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Validation", func() {

	Context("When the Meridio Operator is deployed", func() {
		It("should have the Trench CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: TrenchCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the Attractor CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: AttractorCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the Gateway CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: GatewayCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the VIP CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: VIPCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the Meridio Operator pods in running state", func() {
			operatorName := "meridio-operator-controller-manager"
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(operatorNamespace).Get(context.Background(), operatorName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
			listOptions := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("control-plane=%s", "controller-manager"),
			}
			pods, err := clientset.CoreV1().Pods(operatorNamespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			deployment, err := clientset.AppsV1().Deployments(operatorNamespace).Get(context.Background(), operatorName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(int(deployment.Status.Replicas)))
			for _, pod := range pods.Items {
				Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
			}
		})
	})

})
