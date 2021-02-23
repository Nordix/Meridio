VERSION ?= master

REGISTRY ?= localhost:5000
ORGANIZATION ?= nvip

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

.PHONY: clear
clear:

.PHONY: default
default: 
