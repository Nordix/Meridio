package e2e_test

import (
	"context"
	"testing"
	"time"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	meridioe2eutils "github.com/nordix/meridio/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/scale/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout    = time.Minute * 3
	interval   = time.Second * 2
	namespace  = "red"
	trenchName = "trench-a"
)

var (
	kubeAPIClient client.Client
	clientset     *kubernetes.Clientset
)

func TestE2e(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	myScheme := runtime.NewScheme()
	err := scheme.AddToScheme(myScheme)
	Expect(err).ToNot(HaveOccurred())
	err = apiextensions.AddToScheme(myScheme)
	Expect(err).ToNot(HaveOccurred())
	err = meridiov1alpha1.AddToScheme(myScheme)
	Expect(err).ToNot(HaveOccurred())
	config, err := meridioe2eutils.GetConfig()
	Expect(err).ToNot(HaveOccurred())
	kubeAPIClient, err = client.New(config, client.Options{
		Scheme: myScheme,
	})
	Expect(err).ToNot(HaveOccurred())
	clientset, err = meridioe2eutils.GetClientSet()
	Expect(err).ToNot(HaveOccurred())

	n := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), n, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
})
