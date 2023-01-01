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

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("TAPA", func() {

	var (
		targetPod *v1.Pod
	)

	BeforeEach(func() {
		By(fmt.Sprintf("Selecting the first target from the deployment with label app=%s in namespace %s", config.targetADeploymentName, config.k8sNamespace))
		if targetPod != nil {
			return
		}
		listOptions := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", config.targetADeploymentName),
		}
		pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(pods.Items)).To(BeNumerically(">", 0))
		targetPod = &pods.Items[0]
	})

	Describe("close-open", func() {
		When("Close stream-a-I in one of the target from target-a-deployment-name and re-open it", func() {
			BeforeEach(func() {
				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAI, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAI})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAI to be closed
				By(fmt.Sprintf("Waiting the stream to be closed in pod %s using ./target-client watch", targetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					return len(streamStatus) == 0
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorA1
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
				}
				pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, (numberOfTargetA - 1)))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAI)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == (numberOfTargetA - 1)
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			AfterEach(func() {
				By(fmt.Sprintf("Reopening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAI, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAI})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAI to be opened
				By(fmt.Sprintf("Waiting the stream to be opened in pod %s using ./target-client watch", targetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 1 &&
						streamStatus[0].Status == "OPEN" &&
						streamStatus[0].Trench == config.trenchA &&
						streamStatus[0].Conduit == config.conduitA1 &&
						streamStatus[0].Stream == config.streamAI {
						return true
					}
					return false
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorA1
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
				}
				pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, numberOfTargetA))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAI)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == numberOfTargetA
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA-1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
					_, exists := lastingConnections[targetPod.Name]
					Expect(exists).ToNot(BeTrue(), "The target with the stream closed should have received no traffic")
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA-1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
					_, exists := lastingConnections[targetPod.Name]
					Expect(exists).ToNot(BeTrue(), "The target with the stream closed should have received no traffic")
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("open-second-stream", func() {
		When("Open stream-a-II in one of the target from target-a-deployment-name and close it", func() {

			BeforeEach(func() {
				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAII, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAII to be opened
				By(fmt.Sprintf("Waiting the stream to be opened in pod %s using ./target-client watch", targetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 2 &&
						streamStatus[0].Stream != streamStatus[1].Stream &&
						streamStatus[0].Status == "OPEN" &&
						streamStatus[0].Trench == config.trenchA &&
						streamStatus[0].Conduit == config.conduitA1 &&
						(streamStatus[0].Stream == config.streamAI || streamStatus[0].Stream == config.streamAII) &&
						streamStatus[1].Status == "OPEN" &&
						streamStatus[1].Trench == config.trenchA &&
						streamStatus[1].Conduit == config.conduitA1 &&
						(streamStatus[1].Stream == config.streamAI || streamStatus[1].Stream == config.streamAII) {
						return true
					}
					return false
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorA1
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
				}
				pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, 1))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAII)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == 1
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			AfterEach(func() {
				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAII, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAII to be closed
				By(fmt.Sprintf("Waiting the stream to be closed in pod %s using ./target-client watch", targetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 1 &&
						streamStatus[0].Status == "OPEN" &&
						streamStatus[0].Trench == config.trenchA &&
						streamStatus[0].Conduit == config.conduitA1 &&
						streamStatus[0].Stream == config.streamAI {
						return true
					}
					return false
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorA1
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
				}
				pods, err := clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, numberOfTargetA))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAII)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == 0
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				// test traffic on stream-a-I
				protocol := "tcp"
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}

				// test traffic on stream-a-II
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.flowAYTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAYTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

})
