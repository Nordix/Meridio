# Meridio

<img src="docs/resources/Logo.svg" width="118" height="100">

[![GitHub release](https://img.shields.io/github/release/nordix/meridio)](https://GitHub.com/nordix/meridio/releases/)
[![Build Status](https://jenkins.nordix.org/job/nordix-nsm-meridio-ctraffic-verify-build-master/badge/icon)](https://jenkins.nordix.org/job/nordix-nsm-meridio-ctraffic-verify-build-master/)
[![Go Reference](https://img.shields.io/badge/godoc-reference-blue)](https://pkg.go.dev/github.com/nordix/meridio)
[![go.mod version](https://img.shields.io/github/go-mod/go-version/nordix/meridio)](https://github.com/nordix/meridio)
[![Go Report Card](https://goreportcard.com/badge/github.com/nordix/meridio)](https://goreportcard.com/report/github.com/nordix/meridio)
[![GitHub stars](https://img.shields.io/github/stars/nordix/meridio)](https://github.com/nordix/meridio/stargazers)
![GitHub](https://img.shields.io/github/license/nordix/meridio)

Meridio is an Open Source project providing capabilities to facilitate attraction and distribution of external traffic within Kubernetes. It operates on layer 3/4 to provide traffic distribution via so-called secondary networking upholding separation from the traffic distributed on the default "primary" network within the cluster.

In order to attract traffic towards different services exposed by user applications, service addresses (VIPs) are announced to gateways via different kinds of routing protocols monitored by link-supervision mechanisms available in Meridio.

In addition, Meridio enables development and usage of highly configurable network services thanks to traffic classifiers which allows users to separate the traffic into multiple groups. For now, only a TCP/UDP/SCTP stateless (Maglev) load-balancer is supported as network service.

Through an gRPC API hosted in a sidecar container, user applications can control at runtime external networks and network services attached to the pod, and thus start or stop traffic towards the pod.

![Overview](docs/resources/High-Level-Overview.svg)

## Getting Started

* [High Level Overview](docs/overview.md)
* [Architecture and Concepts](docs/readme.md#table-of-contents)
* [Quick Installation / Demo](docs/demo/readme.md)
* [Deployment](docs/deployment.md)
* [Contributing](CONTRIBUTING.md)
* [Frequently Asked Questions](docs/faq.md)

## Features

### Secondary Networking

Isolation of the traffic and the network is a key aspect for Meridio, it improves **the resiliency, the security, the decongestion and the minimization of the traffic**. In addition, each network can have its own specificities and capabilities provided via **different dataplane methods (VPP, OVS, Kernel, accelerated network...)** which can carry **exotic protocols**. Meridio is for now, providing dataplane only for TCP, UDP and SCTP traffic on IPv4 and IPv6.

### External Traffic Attraction

Announcing service-addresses (VIP-addresses) to the external gateways permits the traffic to be attracted towards the different services exposed by the target applications. Frontend services provide different **connectivity mechanisms such as VLANs or host network interfaces** to connect the network services to the gateways. To **announce the service and ensure the link-supervision**, routing protocols are used. For now, **BGP and Static BFD** are supported.

### Network Services

Development of new network services with more or different capabilities (network acceleration, SCTP load-balancing...) is possible within Meridio.
As the current default network service, a **no-NAT stateless Load-Balancer** is offered by Meridio. It provides traffic classification (based on 5-tuple) to steer traffic into multiple different instances of the network service applications can subscribe to.

### Runtime Configuration

Meridio users have the flexibility to **adjust the network services on the fly** as they desire. Traffic `attractors`, `streams` gathering traffic into logical groups and traffic classifiers (`flows`) can be added, removed and updated without any redeployment of the resources, and with no traffic disturbance. Individually, each user pods have the ability to **select the traffic to consume at runtime** which will produce secondary network reorganization to cover the user pods needs and requests.

## Community

### Slack

The team is reachable on slack for any question, feedback or help: [Slack](https://cloud-native.slack.com/archives/C03ETG3J04S)

### Events

* Cloud Native Telco Day EU 2022 - [Network Service Mesh at Scale for Telco Networking](https://sched.co/zsoW)

## Prerequisites

To run Meridio, the following are required:
* Kubernetes 1.19+
* Spire
* Network Service Mesh 1.3+
