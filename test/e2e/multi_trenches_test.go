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

var _ = Describe("MultiTrenches", func() {

	Context("With two trenches (trench-a and trench-b) deployed in namespace 'red' containing both 2 VIP addresses (20.0.0.1:5000, [2000::1]:5000) and 4 target pods in each trench running ctraffic", func() {

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

		When("traffic is sent on 2 trenches at the same time with the same VIP address", func() {
			var (
				trenchALastingConns map[string]int
				trenchALostConns    map[string]int
				trenchBLastingConns map[string]int
				trenchBLostConns    map[string]int
			)

			BeforeEach(func() {
				trenchADone := make(chan bool)
				var trenchAErr error
				trenchBDone := make(chan bool)
				var trenchBErr error
				go func() {
					trenchALastingConns, trenchALostConns, trenchAErr = utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
					trenchADone <- true
				}()
				go func() {
					trenchBLastingConns, trenchBLostConns, trenchBErr = utils.SendTraffic(trafficGeneratorCMD, trenchB, namespace, ipPort, 400, 100)
					trenchBDone <- true
				}()
				<-trenchADone
				<-trenchBDone
				Expect(trenchAErr).NotTo(HaveOccurred())
				Expect(trenchBErr).NotTo(HaveOccurred())
			})

			It("should be possible to send traffic on the 2 trenches using the same VIP", func() {
				Expect(len(trenchALostConns)).To(Equal(0))
				Expect(len(trenchALastingConns)).To(Equal(4))
				Expect(len(trenchBLostConns)).To(Equal(0))
				Expect(len(trenchBLastingConns)).To(Equal(4))
			})
		})

		When("a target disconnects from a trench and connect to another one", func() {
			BeforeEach(func() {
				_, err := utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "disconnect", "-ns", networkServiceName, "-t", trench})
				Expect(err).NotTo(HaveOccurred())
				_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "connect", "-ns", networkServiceName, "-t", trenchB})
				Expect(err).NotTo(HaveOccurred())
				_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "request", "-ns", networkServiceName, "-t", trenchB})
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, err := utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "connect", "-ns", networkServiceName, "-t", trench})
				Expect(err).NotTo(HaveOccurred())
				_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "request", "-ns", networkServiceName, "-t", trench})
				Expect(err).NotTo(HaveOccurred())
				_, err = utils.PodExec(targetPod, "ctraffic", []string{"./target-client", "disconnect", "-ns", networkServiceName, "-t", trenchB})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should receive the traffic on the other trench", func() {
				By("Verifying trench-a has only 3 targets")
				lastingConn, lostConn, err := utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(lostConn)).To(Equal(0))
				Expect(len(lastingConn)).To(Equal(3))

				By("Verifying trench-b has only 5 targets")
				lastingConn, lostConn, err = utils.SendTraffic(trafficGeneratorCMD, trenchB, namespace, ipPort, 400, 100)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(lostConn)).To(Equal(0))
				Expect(len(lastingConn)).To(Equal(5))
			})
		})

	})
})
