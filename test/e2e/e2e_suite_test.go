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
	"bytes"
	"context"
	"flag"
	"os/exec"
	"testing"
	"time"

	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	trafficGeneratorCMD string
	namespace           string
	script              string

	trenchAName   string
	trenchBName   string
	conduitA1Name string
	conduitB1Name string
	streamA1Name  string
	streamB1Name  string

	tcpIPv4 string
	tcpIPv6 string
	udpIPv4 string
	udpIPv6 string

	newTCPVIP string

	lbfeDeploymentName string

	targetADeploymentName string
	numberOfTargetA       int
	targetBDeploymentName string
	numberOfTargetB       int

	clientset *kubernetes.Clientset

	trafficGeneratorHost *utils.TrafficGeneratorHost
	trafficGenerator     utils.TrafficGenerator
)

const (
	timeout  = time.Minute * 3
	interval = time.Second * 2
)

func init() {
	flag.StringVar(&trafficGeneratorCMD, "traffic-generator-cmd", "docker exec -i {trench}", "Command to use to connect to the traffic generator. All occurences of '{trench}' will be replaced with the trench name.")
	flag.StringVar(&namespace, "namespace", "red", "the namespace where expects operator to exist")
	flag.StringVar(&script, "script", "./data/kind/test.sh", "path + script used by the e2e tests")
	flag.StringVar(&trenchAName, "trench-a-name", "trench-a", "Name of trench-a (see e2e documentation diagram)")
	flag.StringVar(&trenchBName, "trench-b-name", "trench-b", "Name of trench-b (see e2e documentation diagram)")
	flag.StringVar(&conduitA1Name, "conduit-a-1-name", "conduit-a-1", "Name of conduit-a-1 (see e2e documentation diagram)")
	flag.StringVar(&conduitB1Name, "conduit-b-1-name", "conduit-b-1", "Name of conduit-b-1 (see e2e documentation diagram)")
	flag.StringVar(&streamA1Name, "stream-a-1-name", "stream-a-1", "Name of stream-a-1 (see e2e documentation diagram)")
	flag.StringVar(&streamB1Name, "stream-b-1-name", "stream-b-1", "Name of stream-b-1 (see e2e documentation diagram)")
	flag.StringVar(&lbfeDeploymentName, "lb-fe-deployment-name", "lb-fe-attractor-a-1", "Name of load-balancer deployment in trench-a")
	flag.StringVar(&targetADeploymentName, "target-a-deployment-name", "target-a", "Name of target-a deployment in trench-a")
	flag.StringVar(&targetBDeploymentName, "target-b-deployment-name", "target-b", "Name of target-b deployment in trench-b")
	flag.StringVar(&tcpIPv4, "tcp-ipv4", "20.0.0.1:4000", "IP + Port used for testing IPv4 TCP")
	flag.StringVar(&tcpIPv6, "tcp-ipv6", "[2000::1]:4000", "IP + Port used for testing IPv6 TCP")
	flag.StringVar(&udpIPv4, "udp-ipv4", "20.0.0.1:4003", "IP + Port used for testing IPv4 UDP")
	flag.StringVar(&udpIPv6, "udp-ipv6", "[2000::1]:4003", "IP + Port used for testing IPv6 UDP")
	flag.StringVar(&newTCPVIP, "new-tcp-vip", "60.0.0.150:4000", "IP + Port used for testing a new VIP with TCP")
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

	trafficGeneratorHost = &utils.TrafficGeneratorHost{
		TrafficGeneratorCommand: trafficGeneratorCMD,
	}
	trafficGenerator = &utils.MConnect{
		NConn: 400,
	}

	cmd := exec.Command(script, "init")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	Expect(stderr.String()).To(BeEmpty())
	Expect(err).ToNot(HaveOccurred())

	deploymentTargetA, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), targetADeploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	numberOfTargetA = int(*deploymentTargetA.Spec.Replicas)

	deploymentTargetB, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), targetADeploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	numberOfTargetB = int(*deploymentTargetB.Spec.Replicas)
})

var _ = AfterSuite(func() {
	cmd := exec.Command(script, "end")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	Expect(stderr.String()).To(BeEmpty())
	Expect(err).ToNot(HaveOccurred())
})
