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
	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IngressTraffic", func() {

	var (
		lostConnections    int
		lastingConnections map[string]int
		ipPort             string
		protocol           string
	)

	JustBeforeEach(func() {
		lastingConnections, lostConnections = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
	})

	Describe("TCP-IPv4", func() {
		When("Send tcp traffic in trench-a with vip-1-v4 as destination IP and flow-a-z-tcp-destination-port-0 as destination port", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
				protocol = "tcp"
			})
			It("(Traffic) is received by the targets", func() {
				if utils.IsIPv6(config.ipFamily) {
					Skip("The test runs only IPv6")
				}
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})
	})

	Describe("TCP-IPv6", func() {
		When("Send tcp traffic in trench-a with vip-1-v6 as destination IP and flow-a-z-tcp-destination-port-0 as destination port", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
				protocol = "tcp"
			})
			It("(Traffic) is received by the targets", func() {
				if utils.IsIPv4(config.ipFamily) {
					Skip("The test runs only IPv4")
				}
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})
	})

	Describe("UDP-IPv4", func() {
		When("Send udp traffic in trench-a with vip-1-v4 as destination IP and flow-a-z-udp-destination-port-0 as destination port", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V4, config.flowAZUdpDestinationPort0)
				protocol = "udp"
			})
			It("(Traffic) is received by the targets", func() {
				if utils.IsIPv6(config.ipFamily) {
					Skip("The test runs only IPv6")
				}
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})
	})

	Describe("UDP-IPv6", func() {
		When("Send udp traffic in trench-a with vip-1-v6 as destination IP and flow-a-z-udp-destination-port-0 as destination port", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V6, config.flowAZUdpDestinationPort0)
				protocol = "udp"
			})
			It("(Traffic) is received by the targets", func() {
				if utils.IsIPv4(config.ipFamily) {
					Skip("The test runs only IPv4")
				}
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})
	})

})
