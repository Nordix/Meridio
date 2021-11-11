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

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio/test/e2e/operator"
	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = XDescribe("MultiTrenches", func() {

	trenchBResources := getTrenchB()

	BeforeEach(func() {
		if testWithOperator {
			op.DeployMeridio(trenchBResources)
		}
	})

	AfterEach(func() {
		if testWithOperator {
			op.UndeployMerido(trenchBResources)
		}
	})

	Context("With two trenches containing both 2 VIP addresses (20.0.0.1:5000, [2000::1]:5000) and 4 target pods in each trench running ctraffic", func() {

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

func getTrenchB() operator.MeridioResources {
	var m operator.MeridioResources
	trenchBName := "trench-b"
	m.Trench = meridiov1alpha1.Trench{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trenchBName,
			Namespace: namespace,
		},
		// by default ip family is "dual-stack"
	}

	m.Attractor = meridiov1alpha1.Attractor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "attractor-b",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchBName,
			},
		},
		Spec: meridiov1alpha1.AttractorSpec{
			VlanID:         100,
			VlanInterface:  "eth0",
			Gateways:       []string{"gateway3", "gateway4"},
			Vips:           []string{"vip4", "vip5", "vip6"},
			VlanPrefixIPv4: "169.254.100.0/24",
			VlanPrefixIPv6: "100:100::/64",
		},
	}

	m.Conduit = meridiov1alpha1.Conduit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lb-fe",
			Namespace: namespace,
			Labels: map[string]string{
				"trench": trenchBName,
			},
		},
		Spec: meridiov1alpha1.ConduitSpec{
			Replicas: int32pointer(2),
		},
	}

	m.Gateways = []meridiov1alpha1.Gateway{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gateway1",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.GatewaySpec{
				Address: "169.254.100.150",
				Bgp: meridiov1alpha1.BgpSpec{
					LocalASN:   uint32pointer(8103),
					RemoteASN:  uint32pointer(4248829953),
					LocalPort:  uint16pointer(10179),
					RemotePort: uint16pointer(10179),
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gateway2",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.GatewaySpec{
				Address: "100:100::150",
				Bgp: meridiov1alpha1.BgpSpec{
					LocalASN:   uint32pointer(8103),
					RemoteASN:  uint32pointer(4248829953),
					LocalPort:  uint16pointer(10179),
					RemotePort: uint16pointer(10179),
				},
			},
		},
	}

	m.Vips = []meridiov1alpha1.Vip{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vip1",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.VipSpec{
				Address: "20.0.0.1/32",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vip2",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.VipSpec{
				Address: "2000::1/128",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vip3",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.VipSpec{
				Address: "40.0.0.0/24",
			},
		},
	}

	m.Streams = []meridiov1alpha1.Stream{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stream-a",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.StreamSpec{
				Conduit: "lb-fe",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stream-b",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.StreamSpec{
				Conduit: "lb-fe",
			},
		},
	}

	m.Flows = []meridiov1alpha1.Flow{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flow-a",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.FlowSpec{
				Vips:             []string{"vip1", "vip2"},
				SourceSubnets:    []string{"0.0.0.0/0", "0:0:0:0:0:0:0:0/0"},
				DestinationPorts: []string{"5000"},
				SourcePorts:      []string{"1024-65535"},
				Protocols:        []string{"tcp"},
				Stream:           "stream-a",
				Priority:         1,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flow-b",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trenchBName,
				},
			},
			Spec: meridiov1alpha1.FlowSpec{
				Vips:             []string{"vip3"},
				SourceSubnets:    []string{"0.0.0.0/0", "0:0:0:0:0:0:0:0/0"},
				DestinationPorts: []string{"5000"},
				SourcePorts:      []string{"1024-65535"},
				Protocols:        []string{"tcp"},
				Stream:           "stream-b",
				Priority:         1,
			},
		},
	}
	return m
}
