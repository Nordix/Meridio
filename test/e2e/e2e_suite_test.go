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
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	clientset *kubernetes.Clientset

	trafficGeneratorHost *utils.TrafficGeneratorHost
	trafficGenerator     utils.TrafficGenerator

	numberOfTargetA int
	numberOfTargetB int

	config *e2eTestConfiguration

	focusString string
	skipString  string
)

type e2eTestConfiguration struct {
	trafficGeneratorCMD string
	script              string
	logCollectorEnabled bool

	k8sNamespace              string
	targetADeploymentName     string
	trenchA                   string
	attractorA1               string
	conduitA1                 string
	streamAI                  string
	streamAII                 string
	flowAZTcpDestinationPort0 int
	flowAZUdpDestinationPort0 int
	flowAYTcpDestinationPort0 int
	vip1V4                    string
	vip1V6                    string
	targetBDeploymentName     string
	trenchB                   string
	conduitB1                 string
	streamBI                  string
	vip2V4                    string
	vip2V6                    string
	streamAIII                string
	flowAXTcpDestinationPort0 int
	conduitA2                 string
	streamAIV                 string
	flowAWTcpDestinationPort0 int
	vip3V4                    string
	vip3V6                    string
	conduitA3                 string

	statelessLbFeDeploymentNameAttractorA1 string
	statelessLbFeDeploymentNameAttractorB1 string
	statelessLbFeDeploymentNameAttractorA2 string
	statelessLbFeDeploymentNameAttractorA3 string
	ipFamily                               string
}

const (
	eventuallyTimeout  = time.Minute * 3
	eventuallyInterval = time.Second * 2
	timeoutTest        = time.Minute * 15
)

func init() {
	config = &e2eTestConfiguration{}
	flag.StringVar(&config.trafficGeneratorCMD, "traffic-generator-cmd", "", "Command to use to connect to the traffic generator. All occurences of '{trench}' will be replaced with the trench name.")
	flag.StringVar(&config.script, "script", "", "Path + script used by the e2e tests")
	flag.StringVar(&skipString, "skip", "", "Skip specific tests")
	flag.StringVar(&focusString, "focus", "", "Focus on specific tests")
	flag.BoolVar(&config.logCollectorEnabled, "log-collector-enabled", true, "Is log collector enabled")

	flag.StringVar(&config.k8sNamespace, "k8s-namespace", "", "Name of the namespace")
	flag.StringVar(&config.targetADeploymentName, "target-a-deployment-name", "", "Name of the namespace")
	flag.StringVar(&config.trenchA, "trench-a", "", "Name of the trench")
	flag.StringVar(&config.attractorA1, "attractor-a-1", "", "Name of the attractor")
	flag.StringVar(&config.conduitA1, "conduit-a-1", "", "Name of the conduit")
	flag.StringVar(&config.streamAI, "stream-a-I", "", "Name of the stream")
	flag.StringVar(&config.streamAII, "stream-a-II", "", "Name of the stream")
	flag.IntVar(&config.flowAZTcpDestinationPort0, "flow-a-z-tcp-destination-port-0", 4000, "Destination port 0")
	flag.IntVar(&config.flowAZUdpDestinationPort0, "flow-a-z-udp-destination-port-0", 4000, "Destination port 0")
	flag.IntVar(&config.flowAYTcpDestinationPort0, "flow-a-y-tcp-destination-port-0", 4000, "Destination port 0")
	flag.StringVar(&config.vip1V4, "vip-1-v4", "", "Address of the vip v4 number 1")
	flag.StringVar(&config.vip1V6, "vip-1-v6", "", "Address of the vip v6 number 1")
	flag.StringVar(&config.targetBDeploymentName, "target-b-deployment-name", "", "Name of the target deployment")
	flag.StringVar(&config.trenchB, "trench-b", "", "Name of the trench")
	flag.StringVar(&config.conduitB1, "conduit-b-1", "", "Name of the conduit")
	flag.StringVar(&config.streamBI, "stream-b-I", "", "Name of the stream")
	flag.StringVar(&config.vip2V4, "vip-2-v4", "", "Address of the vip v4 number 2")
	flag.StringVar(&config.vip2V6, "vip-2-v6", "", "Address of the vip v6 number 2")
	flag.StringVar(&config.streamAIII, "stream-a-III", "", "Name of the stream")
	flag.IntVar(&config.flowAXTcpDestinationPort0, "flow-a-x-tcp-destination-port-0", 4000, "Destination port 0")
	flag.StringVar(&config.conduitA2, "conduit-a-2", "", "Name of the conduit")
	flag.StringVar(&config.streamAIV, "stream-a-IV", "", "Name of the stream")
	flag.IntVar(&config.flowAWTcpDestinationPort0, "flow-a-w-tcp-destination-port-0", 4000, "Destination port 0")
	flag.StringVar(&config.vip3V4, "vip-3-v4", "", "Address of the vip v4 number 3")
	flag.StringVar(&config.vip3V6, "vip-3-v6", "", "Address of the vip v6 number 3")
	flag.StringVar(&config.conduitA3, "conduit-a-3", "", "Name of the conduit")

	flag.StringVar(&config.statelessLbFeDeploymentNameAttractorA1, "stateless-lb-fe-deployment-name-attractor-a-1", "", "Name of stateless-lb-fe deployment in attractor-a-1")
	flag.StringVar(&config.statelessLbFeDeploymentNameAttractorB1, "stateless-lb-fe-deployment-name-attractor-b-1", "", "Name of stateless-lb-fe deployment in attractor-b-1")
	flag.StringVar(&config.statelessLbFeDeploymentNameAttractorA2, "stateless-lb-fe-deployment-name-attractor-a-2", "", "Name of stateless-lb-fe deployment in attractor-a-2")
	flag.StringVar(&config.statelessLbFeDeploymentNameAttractorA3, "stateless-lb-fe-deployment-name-attractor-a-3", "", "Name of stateless-lb-fe deployment in attractor-a-3")
	flag.StringVar(&config.ipFamily, "ip-family", "", "IP Family")
}

func TestE2e(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	RegisterFailHandler(onFailure)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.SkipStrings = cleanSlice(append(suiteConfig.SkipStrings, strings.Split(skipString, ",")...))
	suiteConfig.FocusStrings = cleanSlice(append(suiteConfig.FocusStrings, strings.Split(focusString, ",")...))
	RunSpecs(t, "E2e Suite", suiteConfig, reporterConfig)
}

// removes spaces in entries and removes empty entries
func cleanSlice(items []string) []string {
	res := []string{}
	for _, item := range items {
		i := strings.ReplaceAll(item, " ", "")
		if i == "" {
			continue
		}
		res = append(res, i)
	}
	return res
}

var _ = BeforeSuite(func() {
	var err error
	clientset, err = utils.GetClientSet()
	Expect(err).ToNot(HaveOccurred())

	trafficGeneratorHost = &utils.TrafficGeneratorHost{
		TrafficGeneratorCommand: config.trafficGeneratorCMD,
	}
	trafficGenerator = &utils.MConnect{
		NConn: 400,
	}

	err = utils.Exec(config.script, "init")
	Expect(err).ToNot(HaveOccurred())

	deploymentTargetA, err := clientset.AppsV1().Deployments(config.k8sNamespace).Get(context.Background(), config.targetADeploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	numberOfTargetA = int(*deploymentTargetA.Spec.Replicas)

	deploymentTargetB, err := clientset.AppsV1().Deployments(config.k8sNamespace).Get(context.Background(), config.targetADeploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	numberOfTargetB = int(*deploymentTargetB.Spec.Replicas)
})

var _ = AfterSuite(func() {
	err := utils.Exec(config.script, "end")
	Expect(err).ToNot(HaveOccurred())
})

func onFailure(message string, callerSkip ...int) {
	By(fmt.Sprintf("Handling the error, collecting the logs: %t", config.logCollectorEnabled))
	if config.logCollectorEnabled {
		_ = utils.Exec(config.script, "on_failure", strconv.FormatInt(CurrentSpecReport().StartTime.UnixNano(), 10))
	}
	Fail(message, callerSkip...)
}
