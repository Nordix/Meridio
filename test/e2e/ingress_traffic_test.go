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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IngressTraffic", func() {

	Context("With one trench containing a stream with 2 VIP addresses and 4 target pods running", func() {

		var (
			lostConnections    int
			lastingConnections map[string]int
			vip                string
		)

		JustBeforeEach(func() {
			var err error
			ipPort := fmt.Sprintf("%s:%s", vip, port)

			lastingConnections, lostConnections = trafficGeneratorHost.SendTraffic(trafficGenerator, trenchAName, namespace, ipPort)
			Expect(err).NotTo(HaveOccurred())
		})

		When("sending traffic to a registered IPv4", func() {
			BeforeEach(func() {
				vip = ipv4
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargets))
			})
		})

		When("sending traffic to a registered IPv6", func() {
			BeforeEach(func() {
				vip = ipv6
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargets))
			})
		})

	})

})
