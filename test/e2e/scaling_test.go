/*
Copyright (c) 2021-2022 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Scaling", func() {

	var (
		replicas int
		scale    *autoscalingv1.Scale
	)

	BeforeEach(func() {
		replicas = numberOfTargetA
		scale = &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.targetADeploymentName,
				Namespace: config.k8sNamespace,
			},
			Spec: autoscalingv1.ScaleSpec{
				Replicas: int32(replicas),
			},
		}
	})

	JustBeforeEach(func() {
		scale.Spec.Replicas = int32(replicas)

		listOptions := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", config.targetADeploymentName),
		}
		pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Current list of targets: %s", podListToString(pods)))

		By(fmt.Sprintf("Scaling %s deployment to %d", config.targetADeploymentName, int(scale.Spec.Replicas)))
		_, err = clientset.AppsV1().Deployments(config.k8sNamespace).UpdateScale(context.Background(), config.targetADeploymentName, scale, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// wait for all targets to be in Running mode
		By(fmt.Sprintf("Waiting for the deployment %s to be scaled to %d", config.targetADeploymentName, int(scale.Spec.Replicas)))
		Eventually(func() bool {
			deployment, err := clientset.AppsV1().Deployments(config.k8sNamespace).Get(context.Background(), config.targetADeploymentName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
			if err != nil {
				return false
			}
			return len(pods.Items) == int(scale.Spec.Replicas) && deployment.Status.ReadyReplicas == deployment.Status.Replicas
		}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		By(fmt.Sprintf("Current list of targets: %s", podListToString(pods)))

		// wait for all identifiers to be in NFQLB
		listOptions = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
		}
		pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
		Expect(err).NotTo(HaveOccurred())
		for _, pod := range pods.Items {
			By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, int(scale.Spec.Replicas)))
			Eventually(func() bool {
				nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAI)})
				Expect(err).NotTo(HaveOccurred())
				return utils.ParseNFQLB(nfqlbOutput) == int(scale.Spec.Replicas)
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		}
	})

	AfterEach(func() {
		scale.Spec.Replicas = int32(numberOfTargetA)

		listOptions := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", config.targetADeploymentName),
		}
		pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("Current list of targets: %s", podListToString(pods)))

		// scale
		By(fmt.Sprintf("Scaling %s deployment to %d", config.targetADeploymentName, int(scale.Spec.Replicas)))
		_, err = clientset.AppsV1().Deployments(config.k8sNamespace).UpdateScale(context.Background(), config.targetADeploymentName, scale, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// wait for all targets to be in Running mode
		By(fmt.Sprintf("Waiting for the deployment %s to be scaled to %d", config.targetADeploymentName, int(scale.Spec.Replicas)))
		Eventually(func() bool {
			deployment, err := clientset.AppsV1().Deployments(config.k8sNamespace).Get(context.Background(), config.targetADeploymentName, metav1.GetOptions{})
			if err != nil {
				return false
			}
			pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
			if err != nil {
				return false
			}
			return len(pods.Items) == int(scale.Spec.Replicas) && deployment.Status.ReadyReplicas == deployment.Status.Replicas
		}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		By(fmt.Sprintf("Current list of targets: %s", podListToString(pods)))

		// wait for all identifiers to be in NFQLB
		listOptions = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
		}
		pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
		Expect(err).NotTo(HaveOccurred())
		for _, pod := range pods.Items {
			By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, int(scale.Spec.Replicas)))
			Eventually(func() bool {
				nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAI)})
				Expect(err).NotTo(HaveOccurred())
				return utils.ParseNFQLB(nfqlbOutput) == int(scale.Spec.Replicas)
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		}
	})

	Describe("Scale-Down", func() {
		When("Scale down target-a-deployment-name", func() {
			BeforeEach(func() {
				replicas = numberOfTargetA - 1
			})
			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.tcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(replicas), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.tcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(replicas), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("Scale-Up", func() {
		When("Scale up target-a-deployment-name", func() {
			BeforeEach(func() {
				replicas = numberOfTargetA + 1
			})
			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.tcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(replicas), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.tcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(replicas), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

})

func podListToString(pods *v1.PodList) string {
	res := []string{}
	for _, pod := range pods.Items {
		res = append(res, pod.Name)
	}
	return strings.Join(res, " ")
}
