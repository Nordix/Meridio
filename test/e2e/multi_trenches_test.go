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

var _ = Describe("MultiTrenches", func() {

	Describe("MT-Parallel", func() {
		When("Send traffic in trench-a and trench-b at the same time", func() {
			var (
				trenchALastingConnsV4 map[string]int
				trenchALostConnsV4    int
				trenchBLastingConnsV4 map[string]int
				trenchBLostConnsV4    int
				trenchALastingConnsV6 map[string]int
				trenchALostConnsV6    int
				trenchBLastingConnsV6 map[string]int
				trenchBLostConnsV6    int
			)

			BeforeEach(func() {
				trenchADone := make(chan bool)
				trenchBDone := make(chan bool)
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					go func() {
						ipPort := utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
						protocol := "tcp"
						By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
						trenchALastingConnsV4, trenchALostConnsV4 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
						trenchADone <- true
					}()
					go func() {
						ipPort := utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
						protocol := "tcp"
						By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchB, config.k8sNamespace, ipPort))
						trenchBLastingConnsV4, trenchBLostConnsV4 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, ipPort, protocol)
						trenchBDone <- true
					}()
					<-trenchADone
					<-trenchBDone
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					go func() {
						ipPort := utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
						protocol := "tcp"
						By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
						trenchALastingConnsV6, trenchALostConnsV6 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
						trenchADone <- true
					}()
					go func() {
						ipPort := utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
						protocol := "tcp"
						By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchB, config.k8sNamespace, ipPort))
						trenchBLastingConnsV6, trenchBLostConnsV6 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, ipPort, protocol)
						trenchBDone <- true
					}()
					<-trenchADone
					<-trenchBDone
				}
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					Expect(trenchALostConnsV4).To(Equal(0), "There should be no lost connection: %v", trenchALastingConnsV4)
					Expect(len(trenchALastingConnsV4)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", trenchALastingConnsV4)
					Expect(trenchBLostConnsV4).To(Equal(0), "There should be no lost connection: %v", trenchBLastingConnsV4)
					Expect(len(trenchBLastingConnsV4)).To(Equal(numberOfTargetB), "All targets with the stream opened should have received traffic: %v", trenchBLastingConnsV4)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					Expect(trenchALostConnsV6).To(Equal(0), "There should be no lost connection: %v", trenchALastingConnsV6)
					Expect(len(trenchALastingConnsV6)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", trenchALastingConnsV6)
					Expect(trenchBLostConnsV6).To(Equal(0), "There should be no lost connection: %v", trenchBLastingConnsV6)
					Expect(len(trenchBLastingConnsV6)).To(Equal(numberOfTargetB), "All targets with the stream opened should have received traffic: %v", trenchBLastingConnsV6)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("MT-Switch", func() {
		When("Disconnect a target from target-a-deployment-name from trench-a and connect it to trench-b", func() {
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

				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAI, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAI})
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamBI, config.conduitB1, config.trenchB, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchB, "-c", config.conduitB1, "-s", config.streamBI})
				Expect(err).NotTo(HaveOccurred())

				// wait target have only trenchB/conduitB1/streamBI opened
				By(fmt.Sprintf("Waiting the stream %s (conduit: %s, trench: %s) to be closed and stream %s (conduit: %s, trench: %s) to be opened in pod %s using ./target-client watch",
					config.streamAI, config.conduitA1, config.trenchA,
					config.streamBI, config.conduitB1, config.trenchB,
					targetPod.Name,
				))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 1 && streamStatus[0].Status == "OPEN" && streamStatus[0].Trench == config.trenchB && streamStatus[0].Conduit == config.conduitB1 && streamStatus[0].Stream == config.streamBI {
						return true
					}
					return false
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorA1
				listOptions = metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorA1),
				}
				pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, (numberOfTargetA - 1)))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamAI)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == (numberOfTargetA - 1)
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorB1
				listOptions = metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorB1),
				}
				pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, (numberOfTargetB + 1)))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamBI)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == (numberOfTargetB + 1)
					}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
				}
			})

			AfterEach(func() {
				By(fmt.Sprintf("Closing stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamBI, config.conduitB1, config.trenchB, targetPod.Name, targetPod.Namespace))
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchB, "-c", config.conduitB1, "-s", config.streamBI})
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Opening stream %s (conduit: %s, trench: %s) in target %s in namespace %s", config.streamAI, config.conduitA1, config.trenchA, targetPod.Name, targetPod.Namespace))
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAI})
				Expect(err).NotTo(HaveOccurred())

				// wait target have only trenchA/conduitA1/streamAI opened
				By(fmt.Sprintf("Waiting the stream %s (conduit: %s, trench: %s) to be closed and stream %s (conduit: %s, trench: %s) to be opened in pod %s using ./target-client watch",
					config.streamBI, config.conduitB1, config.trenchB,
					config.streamAI, config.conduitA1, config.trenchA,
					targetPod.Name,
				))
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 1 && streamStatus[0].Status == "OPEN" && streamStatus[0].Trench == config.trenchA && streamStatus[0].Conduit == config.conduitA1 && streamStatus[0].Stream == config.streamAI {
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

				// wait for all identifiers to be in NFQLB in statelessLbFeDeploymentNameAttractorB1
				listOptions = metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", config.statelessLbFeDeploymentNameAttractorB1),
				}
				pods, err = clientset.CoreV1().Pods(config.k8sNamespace).List(context.Background(), listOptions)
				Expect(err).NotTo(HaveOccurred())
				for _, pod := range pods.Items {
					By(fmt.Sprintf("Waiting for nfqlb in the %s (%s) to have %d targets configured", pod.Name, pod.Namespace, numberOfTargetB))
					Eventually(func() bool {
						nfqlbOutput, err := utils.PodExec(&pod, "stateless-lb", []string{"nfqlb", "show", fmt.Sprintf("--shm=tshm-%v", config.streamBI)})
						Expect(err).NotTo(HaveOccurred())
						return utils.ParseNFQLB(nfqlbOutput) == numberOfTargetB
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
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA-1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}

				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchB, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetB+1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchB, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, ipPort, protocol)
					Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					Expect(len(lastingConnections)).To(Equal(numberOfTargetB+1), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

})
