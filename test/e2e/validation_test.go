package e2e_test

import (
	"context"

	"github.com/nordix/meridio-operator/test/e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Validation", func() {

	Context("When the Meridio Operator is deployed", func() {
		It("should have the Trench CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: utils.TrenchCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the Attractor CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: utils.AttractorCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the Gateway CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: utils.GatewayCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have the VIP CRD available in the cluster", func() {
			crd := &apiextensions.CustomResourceDefinition{}
			err := kubeAPIClient.Get(context.Background(), client.ObjectKey{Name: utils.VIPCRDName}, crd)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
