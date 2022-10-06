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

	Context("With one trench containing a stream with 2 VIP addresses and 4 target pods running", func() {

		When("creating a new vip and adding it to existing stream and attractor", func() {
			BeforeEach(func() {
				cmd := exec.Command(config.script, "configuration_new_ip")
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				err := cmd.Run()
				Expect(stderr.String()).To(BeEmpty())
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				cmd := exec.Command(config.script, "configuration_new_ip_revert")
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				err := cmd.Run()
				Expect(stderr.String()).To(BeEmpty())
				Expect(err).ToNot(HaveOccurred())
			})

			It("should receive the traffic correctly", func() {
				By("Checking if all targets have receive traffic via the new VIP with no traffic interruption (no lost connection)")
				lastingConnections, lostConnections := trafficGeneratorHost.SendTraffic(trafficGenerator, config.trenchA, config.k8sNamespace, utils.VIPPort(config.vip2V4, config.flowAZTcpDestinationPort0), "tcp")
				Expect(lostConnections).To(Equal(0))
				Expect(len(lastingConnections)).To(Equal(numberOfTargetA))
			})
		})

	})

})
