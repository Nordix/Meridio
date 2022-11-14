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

	var (
		targetPod *v1.Pod
	)

	BeforeEach(func() {
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
						trenchALastingConnsV4, trenchALostConnsV4 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0), "tcp")
						trenchADone <- true
					}()
					go func() {
						trenchBLastingConnsV4, trenchBLostConnsV4 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0), "tcp")
						trenchBDone <- true
					}()
					<-trenchADone
					<-trenchBDone
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					go func() {
						trenchALastingConnsV6, trenchALostConnsV6 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0), "tcp")
						trenchADone <- true
					}()
					go func() {
						trenchBLastingConnsV6, trenchBLostConnsV6 = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0), "tcp")
						trenchBDone <- true
					}()
					<-trenchADone
					<-trenchBDone
				}
			})

			It("(Traffic) is received by the targets", func() {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					By("Checking IPv4")
					Expect(trenchALostConnsV4).To(Equal(0))
					Expect(len(trenchALastingConnsV4)).To(Equal(numberOfTargetA))
					Expect(trenchBLostConnsV4).To(Equal(0))
					Expect(len(trenchBLastingConnsV4)).To(Equal(numberOfTargetB))
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					By("Checking IPv6")
					Expect(trenchALostConnsV6).To(Equal(0))
					Expect(len(trenchALastingConnsV6)).To(Equal(numberOfTargetA))
					Expect(trenchBLostConnsV6).To(Equal(0))
					Expect(len(trenchBLastingConnsV6)).To(Equal(numberOfTargetB))
				}
			})
		})
	})

	Describe("MT-Switch", func() {
		When("Disconnect a target from target-a-deployment-name from trench-a and connect it to trench-b", func() {
			BeforeEach(func() {
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAI})
				Expect(err).NotTo(HaveOccurred())
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchB, "-c", config.conduitB1, "-s", config.streamBI})
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 1 && streamStatus[0].Status == "OPEN" && streamStatus[0].Trench == config.trenchB && streamStatus[0].Conduit == config.conduitB1 && streamStatus[0].Stream == config.streamBI {
						return true
					}
					return false
				}, timeout, interval).Should(BeTrue())
			})

			AfterEach(func() {
				_, err := utils.PodExec(targetPod, "example-target", []string{"./target-client", "close", "-t", config.trenchB, "-c", config.conduitB1, "-s", config.streamBI})
				Expect(err).NotTo(HaveOccurred())
				_, err = utils.PodExec(targetPod, "example-target", []string{"./target-client", "open", "-t", config.trenchA, "-c", config.conduitA1, "-s", config.streamAI})
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() bool {
					targetWatchOutput, err := utils.PodExec(targetPod, "example-target", []string{"timeout", "--preserve-status", "0.5", "./target-client", "watch"})
					Expect(err).NotTo(HaveOccurred())
					streamStatus := utils.ParseTargetWatch(targetWatchOutput)
					if len(streamStatus) == 1 && streamStatus[0].Status == "OPEN" && streamStatus[0].Trench == config.trenchA && streamStatus[0].Conduit == config.conduitA1 && streamStatus[0].Stream == config.streamAI {
						return true
					}
					return false
				}, timeout, interval).Should(BeTrue())
			})

			It("(Traffic) is received by the targets", func() {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					By("Checking IPv4 on trench-a")
					lastingConn, lostConn := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0), "tcp")
					Expect(lostConn).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargetA - 1))
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					By("Checking IPv6 on trench-a")
					lastingConn, lostConn := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0), "tcp")
					Expect(lostConn).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargetA - 1))
				}

				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					By("Checking IPv4 on trench-b")
					lastingConn, lostConn := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0), "tcp")
					Expect(lostConn).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargetB + 1))
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					By("Checking IPv6 on trench-b")
					lastingConn, lostConn := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchB, config.k8sNamespace, utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0), "tcp")
					Expect(lostConn).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargetB + 1))
				}
			})
		})
	})

})
