/*
Copyright (c) 2022 Nordix Foundation

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

var _ = Describe("Stream", func() {

	var (
		targetPod  *v1.Pod
		targetPods []v1.Pod
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
		targetPods = pods.Items
	})

	Describe("new-stream", func() {
		When("Configure stream-a-III in conduit-a-1 with a new flow with tcp, flow-a-x-tcp-destination-port-0 as destination port and vip-1-v4 and vip-1-v6", func() {

			BeforeEach(func() {
				By("Configuring the new stream")
				err := utils.Exec(config.script, "new_stream")
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAIII to be opened
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
						(streamStatus[0].Stream == config.streamAI || streamStatus[0].Stream == config.streamAIII) &&
						streamStatus[1].Status == "OPEN" &&
						streamStatus[1].Trench == config.trenchA &&
						streamStatus[1].Conduit == config.conduitA1 &&
						(streamStatus[1].Stream == config.streamAI || streamStatus[1].Stream == config.streamAIII) {
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
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAIII)})
						return err == nil && utils.ParseNFQLB(nfqlbOutput) == 1
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			AfterEach(func() {
				By("Reverting the configuration of the new stream")
				err := utils.Exec(config.script, "new_stream_revert")
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAIII to be closed
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
					By("Waiting for nfqlb to have removed the stream configuration")
					Eventually(func() bool {
						_, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAIII)})
						return err != nil
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				protocol := "tcp"
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.flowAXTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAXTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("stream-max-targets", func() {
		When("Configure stream-a-III as in new-stream test with the max-targets field set to 1 and 2 targets with stream-a-III opened", func() {

			var (
				secondTargetPod *v1.Pod
			)

			BeforeEach(func() {
				By(fmt.Sprintf("Selecting the second target from the deployment with label app=%s in namespace %s", config.targetADeploymentName, config.k8sNamespace))
				Expect(len(targetPods)).To(BeNumerically(">", 1))
				secondTargetPod = &targetPods[1]

				By("Configuring the new stream with max-targets set to 1")
				err := utils.Exec(config.script, "stream_max_targets")
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAIII to be opened
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
						(streamStatus[0].Stream == config.streamAI || streamStatus[0].Stream == config.streamAIII) &&
						streamStatus[1].Status == "OPEN" &&
						streamStatus[1].Trench == config.trenchA &&
						streamStatus[1].Conduit == config.conduitA1 &&
						(streamStatus[1].Stream == config.streamAI || streamStatus[1].Stream == config.streamAIII) {
						return true
					}
					return false
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA1, config.trenchA, secondTargetPod.Name, secondTargetPod.Namespace))
				_, err = utils.PodExec(secondTargetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAIII to be UNAVAILABLE on second target
				By(fmt.Sprintf("Waiting the stream to be unavailable in pod %s using ./target-client watch", secondTargetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(secondTargetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 2 &&
						streamStatus[0].Stream != streamStatus[1].Stream &&
						((streamStatus[0].Status == "OPEN" && streamStatus[0].Stream == config.streamAI) || (streamStatus[0].Status == "UNAVAILABLE" && streamStatus[0].Stream == config.streamAIII)) &&
						streamStatus[0].Trench == config.trenchA &&
						streamStatus[0].Conduit == config.conduitA1 &&
						((streamStatus[1].Status == "OPEN" && streamStatus[1].Stream == config.streamAI) || (streamStatus[1].Status == "UNAVAILABLE" && streamStatus[1].Stream == config.streamAIII)) &&
						streamStatus[1].Trench == config.trenchA &&
						streamStatus[1].Conduit == config.conduitA1 {
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
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAIII)})
						return err == nil && utils.ParseNFQLB(nfqlbOutput) == 1
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			AfterEach(func() {
				By("Reverting the configuration of the new stream")
				err := utils.Exec(config.script, "stream_max_targets_revert")
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA1, config.trenchA, secondTargetPod.Name, secondTargetPod.Namespace))
				_, err = utils.PodExec(secondTargetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAIII to be closed on second target
				By(fmt.Sprintf("Waiting the stream to be closed in pod %s using ./target-client watch", secondTargetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(secondTargetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
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

				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA1/streamAIII to be closed
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
					By("Waiting for nfqlb to have removed the stream configuration")
					Eventually(func() bool {
						_, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAIII)})
						return err != nil
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				protocol := "tcp"
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.flowAXTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAXTcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

})
