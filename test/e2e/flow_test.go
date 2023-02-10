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
)

var _ = Describe("Flow", func() {

	Describe("new-flow", func() {
		When("Configure a new flow with tcp, tcp-destination-port-2 as destination port and vip-1-v4 and vip-1-v6 in stream-a-I", func() {
			BeforeEach(func() {
				By("Configuring the new flow")
				err := utils.Exec(config.script, "new_flow")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				By("Reverting the configuration of the new flow")
				err := utils.Exec(config.script, "new_flow_revert")
				Expect(err).ToNot(HaveOccurred())
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.tcpDestinationPort2)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					if !config.ignoreLostConnections {
						Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					}
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.tcpDestinationPort2)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					if !config.ignoreLostConnections {
						Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					}
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("flow-priority", func() {
		When("Set priority to 3 and add tcp-destination-port-1 as destination port in flow-a-z-tcp", func() {
			BeforeEach(func() {
				By("Configuring the flow")
				err := utils.Exec(config.script, "flow_priority")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				By("Reverting the configuration of the flow")
				err := utils.Exec(config.script, "flow_priority_revert")
				Expect(err).ToNot(HaveOccurred())
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.tcpDestinationPort1)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					if !config.ignoreLostConnections {
						Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					}
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.tcpDestinationPort1)
					protocol := "tcp"
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					if !config.ignoreLostConnections {
						Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					}
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("flow-destination-ports-range", func() {
		When("Set priority to 3 and set 'tcp-destination-port-0'-'tcp-destination-port-2' as destination port in flow-a-z-tcp", func() {
			BeforeEach(func() {
				By("Configuring the flow")
				err := utils.Exec(config.script, "flow_destination_ports_range")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				By("Reverting the configuration of the flow")
				err := utils.Exec(config.script, "flow_destination_ports_range_revert")
				Expect(err).ToNot(HaveOccurred())
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				for _, port := range []int{config.tcpDestinationPort0, config.tcpDestinationPort1, config.tcpDestinationPort2} {
					if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
						ipPort := utils.VIPPort(config.vip1V4, port)
						protocol := "tcp"
						By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
						lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
						if !config.ignoreLostConnections {
							Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
						}
						Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
					}
					if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
						ipPort := utils.VIPPort(config.vip1V6, port)
						protocol := "tcp"
						By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
						lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
						if !config.ignoreLostConnections {
							Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
						}
						Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
					}
				}
			}, SpecTimeout(timeoutTest))
		})
	})

	Describe("flow-byte-matches", func() {
		When("Add tcp-destination-port-2 to destination ports of flow-a-z-tcp and add a byte-match to allow only tcp-destination-port-2", func() {
			BeforeEach(func() {
				By("Configuring the flow")
				err := utils.Exec(config.script, "flow_byte_matches")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				By("Reverting the configuration of the flow")
				err := utils.Exec(config.script, "flow_byte_matches_revert")
				Expect(err).ToNot(HaveOccurred())
			})

			It("(Traffic) is received by the targets", func(ctx context.Context) {
				protocol := "tcp"
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.tcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, _ := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol, utils.WithTimeout("1s"))
					Expect(len(lastingConnections)).To(Equal(0), "No target should have received the traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.tcpDestinationPort0)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, _ := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol, utils.WithTimeout("1s"))
					Expect(len(lastingConnections)).To(Equal(0), "No target should have received the traffic: %v", lastingConnections)
				}

				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					ipPort := utils.VIPPort(config.vip1V4, config.tcpDestinationPort2)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					if !config.ignoreLostConnections {
						Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					}
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					ipPort := utils.VIPPort(config.vip1V6, config.tcpDestinationPort2)
					By(fmt.Sprintf("Sending %s traffic from the TG %s (%s) to %s", protocol, config.trenchA, config.k8sNamespace, ipPort))
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
					if !config.ignoreLostConnections {
						Expect(lostConnections).To(Equal(0), "There should be no lost connection: %v", lastingConnections)
					}
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA), "All targets with the stream opened should have received traffic: %v", lastingConnections)
				}
			}, SpecTimeout(timeoutTest))
		})
	})
})
