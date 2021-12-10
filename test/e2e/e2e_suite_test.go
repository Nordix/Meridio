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

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio/test/e2e/operator"
	"github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	scalescheme "k8s.io/client-go/scale/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	trafficGeneratorCMD string
	testWithOperator    bool
	namespace           string
	networkServiceName  = "load-balancer"
	stream              = "stream-a"

	clientset *kubernetes.Clientset
	op        operator.Operator
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
	flag.BoolVar(&testWithOperator, "test-with-operator", false, "meridio deployment will be installed and updated by meridio operator")
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

	var trenchA operator.MeridioResources

	if testWithOperator {
		By("checking operator deployment")
		myScheme := runtime.NewScheme()

		Expect(kubescheme.AddToScheme(myScheme)).To(Succeed())
		Expect(scalescheme.AddToScheme(myScheme)).To(Succeed())
		Expect(apiextensions.AddToScheme(myScheme)).To(Succeed())
		Expect(meridiov1alpha1.AddToScheme(myScheme)).To(Succeed())

		config := config.GetConfigOrDie()
		kubeAPIClient, err := client.New(config, client.Options{Scheme: myScheme})
		Expect(err).To(BeNil())
		op = operator.Operator{
			Namespace: namespace,
			Client:    kubeAPIClient,
		}
		// verify operator is in the testing namespace
		op.VerifyDeployment()

		// By("cleaning up CRs")
		// op.CleanUpResource()
		// time.Sleep(5 * time.Second)
		trenchA = op.GetMeridioResoucesByTrench(trenchAName, namespace)
		op.AssertMeridioDeploymentsReady(trenchA)

		// change networkServiceName
		networkServiceName = trenchA.Conduit.ObjectMeta.Name
	}
})
