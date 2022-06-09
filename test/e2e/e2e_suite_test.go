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
	"flag"
	"testing"
	"time"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

var (
	trafficGeneratorCMD string
	namespace           string
	networkServiceName  = "load-balancer"
	stream              = "stream-a"

	clientset *kubernetes.Clientset
)

const (
	timeout  = time.Minute * 3
	interval = time.Second * 2

	trenchAName = "trench-a"
	trenchBName = "trench-b"

	targetDeploymentName = "target-a"
	port                 = "5000"
	ipv4                 = "20.0.0.1"
	ipv6                 = "[2000::1]"
	numberOfTargets      = 4
	ipPort               = "20.0.0.1:5000"
)

func init() {
	flag.StringVar(&trafficGeneratorCMD, "traffic-generator-cmd", "docker exec -i {trench}", "Command to use to connect to the traffic generator. All occurences of '{trench}' will be replaced with the trench name.")
	flag.StringVar(&namespace, "namespace", "red", "the namespace where expects operator to exist")
}

func TestE2e(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	var err error
	clientset, err = utils.GetClientSet()
	Expect(err).ToNot(HaveOccurred())
})
