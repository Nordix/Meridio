/*
Copyright (c) 2021 Nordix Foundation

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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Target", func() {

	Context("With one trench (trench-a) deployed in namespace red containing 2 VIP addresses (20.0.0.1:5000, [2000::1]:5000) and 4 target pods running ctraffic", func() {

		var (
			targetPod *v1.Pod
		)

		BeforeEach(func() {
			if targetPod != nil {
				return
			}
			listOptions := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", targetDeploymentName),
			}
			pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(BeNumerically(">", 0))
			targetPod = &pods.Items[0]
		})

		Describe("Stream", func() {
			When("a target is opening a stream", func() {
				var (
					err error
				)

				BeforeEach(func() {
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "close", "-ns", networkServiceName, "-t", trench})
					Expect(err).NotTo(HaveOccurred())
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "request", "-ns", networkServiceName, "-t", trench})
				})

				It("should be able to receive traffic", func() {
					By("Checking if there is no error")
					Expect(err).NotTo(HaveOccurred())

					By("Checking if the target is receiving the traffic")
					lastingConn, lostConn, err := utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(lostConn)).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargets))

					By("Checking if the target has a new network interface")
					targetHasNetworkInterface, err := utils.PodHasNetworkInterface(targetPod, "ctraffic", "nsc")
					Expect(err).NotTo(HaveOccurred())
					Expect(targetHasNetworkInterface).To(BeTrue())
				})
			})

			When("a target is closing a stream", func() {
				var (
					err error
				)

				BeforeEach(func() {
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "close", "-ns", networkServiceName, "-t", trench})
				})

				AfterEach(func() {
					_, err := utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "request", "-ns", networkServiceName, "-t", trench})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should still be connected to the conduit/trench but should not be able receive traffic", func() {
					By("Checking if there is no error")
					Expect(err).NotTo(HaveOccurred())

					By("Checking if the target is not receiving the traffic")
					lastingConn, lostConn, err := utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(lostConn)).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargets - 1))

					By("Checking if the network interface is still in the target")
					targetHasNetworkInterface, err := utils.PodHasNetworkInterface(targetPod, "ctraffic", "nsc")
					Expect(err).NotTo(HaveOccurred())
					Expect(targetHasNetworkInterface).To(BeTrue())
				})
			})
		})

		Describe("Conduit/Trench", func() {
			When("a target is connecting to a conduit/trench", func() {
				var (
					err error
				)

				BeforeEach(func() {
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "disconnect", "-ns", networkServiceName, "-t", trench})
					Expect(err).NotTo(HaveOccurred())
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "connect", "-ns", networkServiceName, "-t", trench})
				})

				AfterEach(func() {
					_, err := utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "request", "-ns", networkServiceName, "-t", trench})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should have a new network interface", func() {
					By("Checking if there is no error")
					Expect(err).NotTo(HaveOccurred())

					By("Checking if the target has a new network interface")
					targetHasNetworkInterface, err := utils.PodHasNetworkInterface(targetPod, "ctraffic", "nsc")
					Expect(err).NotTo(HaveOccurred())
					Expect(targetHasNetworkInterface).To(BeTrue())
				})
			})

			When("a target is disconnecting from a conduit/trench", func() {
				var (
					err error
				)

				BeforeEach(func() {
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "disconnect", "-ns", networkServiceName, "-t", trench})
				})

				AfterEach(func() {
					_, err := utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "connect", "-ns", networkServiceName, "-t", trench})
					Expect(err).NotTo(HaveOccurred())
					_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "request", "-ns", networkServiceName, "-t", trench})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should no longer have the network interface and should not receive traffic anymore", func() {
					By("Checking if there is no error")
					Expect(err).NotTo(HaveOccurred())

					By("Checking if the target is not receiving the traffic")
					lastingConn, lostConn, err := utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(lostConn)).To(Equal(0))
					Expect(len(lastingConn)).To(Equal(numberOfTargets - 1))

					By("Checking if the target no longer has the new network interface")
					targetHasNetworkInterface, err := utils.PodHasNetworkInterface(targetPod, "ctraffic", "nsc")
					Expect(err).NotTo(HaveOccurred())
					Expect(targetHasNetworkInterface).To(BeFalse())
				})
			})
		})

	})

})
