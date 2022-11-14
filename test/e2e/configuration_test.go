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
	"bytes"
	"os/exec"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configuration", func() {

	Describe("new-vip", func() {
		When("Configure vip-2-v4 and vip-2-v6 in flow-a-z-tcp and attractor-a-1", func() {
			BeforeEach(func() {
				cmd := exec.Command(config.script, "configuration_new_vip")
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				err := cmd.Run()
				Expect(stderr.String()).To(BeEmpty())
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				cmd := exec.Command(config.script, "configuration_new_vip_revert")
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				err := cmd.Run()
				Expect(stderr.String()).To(BeEmpty())
				Expect(err).ToNot(HaveOccurred())
			})

			It("(Traffic) is received by the targets", func() {
				if !utils.IsIPv6(config.ipFamily) { // Don't send traffic with IPv4 if the tests are only IPv6
					By("Checking IPv4")
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip2V4, config.flowAZTcpDestinationPort0), "tcp")
					Expect(lostConnections).To(Equal(0))
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
				}
				if !utils.IsIPv4(config.ipFamily) { // Don't send traffic with IPv6 if the tests are only IPv4
					By("Checking IPv6")
					lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip2V6, config.flowAZTcpDestinationPort0), "tcp")
					Expect(lostConnections).To(Equal(0))
					Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
				}
			})
		})
	})

})
