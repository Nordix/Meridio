#!/bin/bash

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh

git clone https://github.com/Nordix/Meridio.git

cd Meridio

docker pull registry.gitlab.com/lionelj/meridio/kind-host:latest
docker tag registry.gitlab.com/lionelj/meridio/kind-host:latest registry.nordix.org/cloud-native/meridio/kind-host:latest

make -s -C docs/demo/scripts/kind/ KUBERNETES_VERSION="v1.26" KUBERNETES_IP_FAMILY="v1.26" NSM_VERSION="v1.7.1" 

helm install meridio-crds https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-CRDs-v1.0.0.tgz --create-namespace --namespace red
helm install meridio https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-v1.0.0.tgz --create-namespace --namespace red --set registry="registry.gitlab.com" --set repository="lionelj/meridio" --set nsm.repository="lionelj/meridio"

clear
