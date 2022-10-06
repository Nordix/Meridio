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

	Context("With one trench containing a stream with 2 VIP addresses and 4 target pods running", func() {

		var (
			lostConnections    int
			lastingConnections map[string]int
			ipPort             string
			protocol           string
		)

		JustBeforeEach(func() {
			lastingConnections, lostConnections = trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, ipPort, protocol)
		})

		When("sending TCP traffic to an IPv4", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V4, config.flowAZTcpDestinationPort0)
				protocol = "tcp"
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})

		When("sending TCP traffic to an IPv6", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V6, config.flowAZTcpDestinationPort0)
				protocol = "tcp"
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})

		When("sending UDP traffic to an IPv4", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V4, config.flowAZUdpDestinationPort0)
				protocol = "udp"
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})

		When("sending UDP traffic to an IPv6", func() {
			BeforeEach(func() {
				ipPort = utils.VIPPort(config.vip1V6, config.flowAZUdpDestinationPort0)
				protocol = "udp"
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})

	})

})
