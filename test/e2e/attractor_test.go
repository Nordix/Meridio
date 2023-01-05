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

var _ = Describe("Attractor", func() {

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

	Describe("new-attractor-nsm-vlan", func() {
		When("Configure a new attractor with new vips vip-2-v4 and vip-2-v6, gateways, conduit conduit-a-3, stream stream-a-III and flow with tcp and flow-a-z-tcp-destination-port-0 as destination port", func() {
			BeforeEach(func() {
				By("Configuring the new attractor")
				err := utils.Exec(config.script, "new_attractor_nsm_vlan")
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA3, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA3, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA3/streamAIII to be opened
				By(fmt.Sprintf("Waiting the stream to be opened in pod %s using ./target-client watch", targetPod.Name))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 2 &&
						streamStatus[0].Stream != streamStatus[1].Stream &&
						streamStatus[0].Conduit != streamStatus[1].Conduit &&
						streamStatus[0].Status == "OPEN" &&
						streamStatus[0].Trench == config.trenchA &&
						((streamStatus[0].Conduit == config.conduitA1 && streamStatus[0].Stream == config.streamAI) || (streamStatus[0].Conduit == config.conduitA3 && streamStatus[0].Stream == config.streamAIII)) &&
						streamStatus[1].Status == "OPEN" &&
						streamStatus[1].Trench == config.trenchA &&
						((streamStatus[1].Conduit == config.conduitA1 && streamStatus[1].Stream == config.streamAI) || (streamStatus[1].Conduit == config.conduitA3 && streamStatus[1].Stream == config.streamAIII)) {
						return true
					}
					return false
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorA3
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA3),
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
				By("Reverting the configuration of the new attractor")
				err := utils.Exec(config.script, "new_attractor_nsm_vlan_revert")
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAIII, config.conduitA3, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA3, "-s", config.streamAIII})
				Expect(err).NotTo(HaveOccurred())

				// wait trenchA/conduitA3/streamAIII to be closed
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
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip2V4, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip2V6, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

})
