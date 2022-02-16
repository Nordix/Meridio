VERSION ?= latest
VERSION_LOAD_BALANCER ?= $(VERSION)
VERSION_PROXY ?= $(VERSION)
VERSION_TAPA ?= $(VERSION)
VERSION_IPAM ?= $(VERSION)
VERSION_NSP ?= $(VERSION)
VERSION_CTRAFFIC ?= $(VERSION)
VERSION_FRONTEND ?= $(VERSION)

E2E_FOCUS ?= ""
TRAFFIC_GENERATOR_CMD ?= "docker exec -i {trench}"

REGISTRY ?= localhost:5000/meridio

.PHONY: all
all: default

.PHONY: build
build:
	docker build -t $(IMAGE) -f ./build/$(IMAGE)/Dockerfile .
.PHONY: tag
tag:
	docker tag $(IMAGE) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

.PHONY: load-balancer
load-balancer:
	VERSION=$(VERSION_LOAD_BALANCER) IMAGE=load-balancer $(MAKE) build tag push

.PHONY: proxy
proxy:
	VERSION=$(VERSION_PROXY) IMAGE=proxy $(MAKE) build tag push

.PHONY: tapa
tapa:
	VERSION=$(VERSION_TAPA) IMAGE=tapa $(MAKE) build tag push

.PHONY: ipam
ipam:
	VERSION=$(VERSION_IPAM) IMAGE=ipam $(MAKE) build tag push

.PHONY: nsp
nsp:
	VERSION=$(VERSION_NSP) IMAGE=nsp $(MAKE) build tag push

.PHONY: ctraffic
ctraffic:
	VERSION=$(VERSION_CTRAFFIC) IMAGE=ctraffic $(MAKE) build tag push

.PHONY: frontend
frontend:
	VERSION=$(VERSION_FRONTEND) IMAGE=frontend $(MAKE) build tag push

.PHONY: ipam-proto
ipam-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/ipam/**/*.proto

.PHONY: nsp-proto
nsp-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/nsp/**/*.proto 

.PHONY: ambassador-proto
ambassador-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/ambassador/**/*.proto

.PHONY: proto
proto: ipam-proto nsp-proto ambassador-proto

.PHONY: clear
clear:

.PHONY: default
default: load-balancer proxy tapa ipam nsp ctraffic frontend

.PHONY: lint
lint: 
	golangci-lint run ./...

NAMESPACE ?= red
TEST_WITH_OPERATOR ?= false
.PHONY: e2e
e2e: 
	ginkgo -v --focus=$(E2E_FOCUS) ./test/e2e/... -- -traffic-generator-cmd=$(TRAFFIC_GENERATOR_CMD) -namespace=${NAMESPACE} -test-with-operator=${TEST_WITH_OPERATOR}

.PHONY: test
test: 
	go test -race -cover -short ./... 

.PHONY: cover
cover: 
	go test -race -coverprofile cover.out -short ./... 
	go tool cover -html=cover.out -o cover.html

.PHONY: check
check: lint test

.PHONY: mocks
mocks:
	mockgen -source=./pkg/ambassador/tap/types/stream.go -destination=./pkg/ambassador/tap/types/mocks/stream.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/types/conduit.go -destination=./pkg/ambassador/tap/types/mocks/conduit.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/types/trench.go -destination=./pkg/ambassador/tap/types/mocks/trench.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/types/registry.go -destination=./pkg/ambassador/tap/types/mocks/registry.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/trench/factory.go -destination=./pkg/ambassador/tap/trench/mocks/factory.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/conduit/configuration.go -destination=./pkg/ambassador/tap/conduit/mocks/configuration.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/conduit/types.go -destination=./pkg/ambassador/tap/conduit/mocks/types.go -package=mocks
	mockgen -source=./pkg/ambassador/tap/stream/types.go -destination=./pkg/ambassador/tap/stream/mocks/types.go -package=mocks
	mockgen -source=./pkg/nsm/client.go -destination=./pkg/nsm/mocks/client.go -package=mocks
