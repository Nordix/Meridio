module github.com/nordix/meridio

go 1.16

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/edwarnicke/grpcfd v0.1.1
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.5.2
	github.com/google/nftables v0.0.0-20210916140115-16a134723a96
	github.com/jinzhu/now v1.1.3 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/networkservicemesh/api v1.1.0
	github.com/networkservicemesh/sdk v1.1.0
	github.com/networkservicemesh/sdk-sriov v1.1.0
	github.com/nordix/meridio-operator v0.0.0-20211110154001-ee8264246a47
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spiffe/go-spiffe/v2 v2.0.0-beta.10
	github.com/stretchr/testify v1.7.0
	github.com/vishvananda/netlink v1.1.0
	go.uber.org/goleak v1.1.10
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/sqlite v1.2.6
	gorm.io/gorm v1.22.3
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	sigs.k8s.io/controller-runtime v0.9.6
)
