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
	"time"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Scaling", func() {

	Context("When trench 'trench-a' is deployed in namespace 'red' with 2 VIP addresses (20.0.0.1:5000, [2000::1]:5000) and 4 target pods running ctraffic", func() {

		var (
			replicas int
			scale    *autoscalingv1.Scale
		)

		BeforeEach(func() {
			replicas = numberOfTargets
			scale = &autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{
					Name:      targetDeploymentName,
					Namespace: namespace,
				},
				Spec: autoscalingv1.ScaleSpec{
					Replicas: int32(replicas),
				},
			}
		})

		JustBeforeEach(func() {
			scale.Spec.Replicas = int32(replicas)
			_, err := clientset.AppsV1().Deployments(namespace).UpdateScale(context.Background(), targetDeploymentName, scale, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), targetDeploymentName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", targetDeploymentName),
				}
				pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
				if err != nil {
					return false
				}
				return len(pods.Items) == int(deployment.Status.Replicas) && deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
		})

		AfterEach(func() {
			scale.Spec.Replicas = numberOfTargets
			_, err := clientset.AppsV1().Deployments(namespace).UpdateScale(context.Background(), targetDeploymentName, scale, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), targetDeploymentName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				listOptions := metav1.ListOptions{
					LabelSelector: fmt.Sprintf("app=%s", targetDeploymentName),
				}
				pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)
				if err != nil {
					return false
				}
				return len(pods.Items) == int(deployment.Status.Replicas) && deployment.Status.ReadyReplicas == deployment.Status.Replicas
			}, timeout, interval).Should(BeTrue())
		})

		When("scaling targets down", func() {
			BeforeEach(func() {
				replicas = 3
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				lastingConnections, lostConnections, err := utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(lostConnections)).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(replicas))
			})
		})

		When("scaling targets up", func() {
			BeforeEach(func() {
				replicas = 5
			})
			It("should receive the traffic correctly", func() {
				By("Waiting for the new targets to be registered")
				time.Sleep(20 * time.Second)
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				lastingConnections, lostConnections, err := utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(lostConnections)).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(replicas))
			})
		})

	})

})
