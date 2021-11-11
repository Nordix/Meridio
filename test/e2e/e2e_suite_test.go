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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

const (
	timeout              = time.Minute * 3
	interval             = time.Second * 2
	trench               = "trench-a"
	targetDeploymentName = "target-a"
	port                 = "5000"
	ipv4                 = "20.0.0.1"
	ipv6                 = "[2000::1]"
	numberOfTargets      = 4
	ipPort               = "20.0.0.1:5000"
	trenchB              = "trench-b"
)

var (
	clientset *kubernetes.Clientset
	op        operator.Operator
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

		trenchA.Trench = meridiov1alpha1.Trench{
			ObjectMeta: metav1.ObjectMeta{
				Name:      trench,
				Namespace: namespace,
			},
			// by default ip family is "dual-stack"
		}

		trenchA.Attractor = meridiov1alpha1.Attractor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "attractor-a",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trench,
				},
			},
			Spec: meridiov1alpha1.AttractorSpec{
				VlanID:         100,
				VlanInterface:  "eth0",
				Gateways:       []string{"gateway-v4-a", "gateway-v6-a"},
				Vips:           []string{"vip-v4-a", "vip-v6-a"},
				VlanPrefixIPv4: "169.254.100.0/24",
				VlanPrefixIPv6: "100:100::/64",
			},
		}

		trenchA.Conduit = meridiov1alpha1.Conduit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lb-fe",
				Namespace: namespace,
				Labels: map[string]string{
					"trench": trench,
				},
			},
			Spec: meridiov1alpha1.ConduitSpec{
				Replicas: int32pointer(2),
			},
		}

		trenchA.Gateways = []meridiov1alpha1.Gateway{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gateway-v4-a",
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trench,
					},
				},
				Spec: meridiov1alpha1.GatewaySpec{
					Address: "169.254.100.150",
					Bgp: meridiov1alpha1.BgpSpec{
						LocalASN:   uint32pointer(8103),
						RemoteASN:  uint32pointer(4248829953),
						HoldTime:   "24s",
						LocalPort:  uint16pointer(10179),
						RemotePort: uint16pointer(10179),
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gateway-v6-a",
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trench,
					},
				},
				Spec: meridiov1alpha1.GatewaySpec{
					Address: "100:100::150",
					Bgp: meridiov1alpha1.BgpSpec{
						LocalASN:   uint32pointer(8103),
						RemoteASN:  uint32pointer(4248829953),
						HoldTime:   "24s",
						LocalPort:  uint16pointer(10179),
						RemotePort: uint16pointer(10179),
					},
				},
			},
		}

		trenchA.Vips = []meridiov1alpha1.Vip{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vip-v4-a",
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trench,
					},
				},
				Spec: meridiov1alpha1.VipSpec{
					Address: "20.0.0.1/32",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vip-v4-6",
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trench,
					},
				},
				Spec: meridiov1alpha1.VipSpec{
					Address: "2000::1/128",
				},
			},
		}

		trenchA.Streams = []meridiov1alpha1.Stream{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "stream-a",
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trench,
					},
				},
				Spec: meridiov1alpha1.StreamSpec{
					Conduit: "lb-fe",
				},
			},
		}

		trenchA.Flows = []meridiov1alpha1.Flow{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "flow-a",
					Namespace: namespace,
					Labels: map[string]string{
						"trench": trench,
					},
				},
				Spec: meridiov1alpha1.FlowSpec{
					Vips:             []string{"vip-v4-a", " vip-v6-a"},
					SourceSubnets:    []string{"0.0.0.0/0", "0:0:0:0:0:0:0:0/0"},
					DestinationPorts: []string{"5000"},
					SourcePorts:      []string{"1024-65535"},
					Protocols:        []string{"tcp"},
					Stream:           "stream-a",
					Priority:         1,
				},
			},
		}
		op.AssertMeridioDeploymentsReady(trenchA)

		// change networkServiceName
		networkServiceName = trenchA.Conduit.ObjectMeta.Name
	}
})

func int32pointer(i int32) *int32 {
	var ret = new(int32)
	*ret = i
	return ret
}

func uint32pointer(i uint32) *uint32 {
	var ret = new(uint32)
	*ret = i
	return ret
}

func uint16pointer(i uint16) *uint16 {
	var ret = new(uint16)
	*ret = i
	return ret
}
