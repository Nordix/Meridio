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
	"fmt"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IngressTraffic", func() {

	Context("When trench 'trench-a' is deployed in namespace 'red' with 2 VIP addresses (20.0.0.1:5000, [2000::1]:5000) and 4 target pods running ctraffic", func() {

		var (
			lostConnections    map[string]int
			lastingConnections map[string]int
			vip                string
		)

		JustBeforeEach(func() {
			var err error
			ipPort := fmt.Sprintf("%s:%s", vip, port)
			lastingConnections, lostConnections, err = utils.SendTraffic(trafficGeneratorCMD, trench, namespace, ipPort, 400, 100)
			Expect(err).NotTo(HaveOccurred())
		})

		When("sending traffic to a registered IPv4", func() {
			BeforeEach(func() {
				vip = ipv4
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(len(lostConnections)).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(4))
			})
		})

		When("sending traffic to a registered IPv6", func() {
			BeforeEach(func() {
				vip = ipv6
			})
			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic with no traffic interruption (no lost connection)")
				Expect(len(lostConnections)).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(4))
			})
		})

	})

})
