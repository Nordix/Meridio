#!/bin/bash

function chart_meridio () {
    rm -rf "$MERIDIO_HELM_PATH"
    mkdir -p $MERIDIO_HELM_PATH
    mkdir -p $MERIDIO_HELM_PATH/templates

    echo """---
apiVersion: v1
description: 'Meridio Operator Chart'
appVersion: '$VERSION'
name: Meridio
version: '$HELM_CHART_VERSION'""" > $MERIDIO_HELM_PATH/Chart.yaml

    echo """---
registry: \"$REGISTRY\"
repository: \"$REPOSITORY\"
nsm:
  repository: \"$NSM_REPOSITORY\"""" > $MERIDIO_HELM_PATH/values.yaml

    make -s print-manifests OPERATOR_NAMESPACE="{{.Release.Namespace}}" VERSION="$VERSION" REGISTRY="{{.Values.registry}}/{{.Values.repository}}" NSM_REPOSITORY="{{.Values.nsm.repository}}" | yq '. | select(.kind != "Namespace" and .kind != "CustomResourceDefinition")' -s "\"$MERIDIO_HELM_PATH/templates/\" + .metadata.name + \"-\" + .kind"
    sed -i '0,/  name: .*/s/^  name: .*/  name: meridio-operator-validating-webhook-configuration-{{.Release.Namespace}}/1' $MERIDIO_HELM_PATH/templates/meridio-operator-validating-webhook-configuration-ValidatingWebhookConfiguration.yml
    helm package $MERIDIO_HELM_PATH --version $HELM_CHART_VERSION --destination $HELM_CHART_PATH
}

function chart_meridio_crds () {
    rm -rf "$MERIDIO_CRDS_HELM_PATH"
    mkdir -p $MERIDIO_CRDS_HELM_PATH
    mkdir -p $MERIDIO_CRDS_HELM_PATH/templates

    echo """---
apiVersion: v1
description: 'Meridio CRDs Chart'
appVersion: '$VERSION'
name: Meridio-CRDs
version: '$HELM_CHART_VERSION'""" > $MERIDIO_CRDS_HELM_PATH/Chart.yaml

    make -s print-manifests OPERATOR_NAMESPACE="{{.Release.Namespace}}" VERSION="$VERSION" | yq '. | select(.kind == "CustomResourceDefinition")' -s "\"$MERIDIO_CRDS_HELM_PATH/templates/\" + .metadata.name"
    helm package $MERIDIO_CRDS_HELM_PATH --version $HELM_CHART_VERSION --destination $HELM_CHART_PATH
}

function chart_meridio_target () {
    cp -r $MERIDIO_TARGET_HELM_CURRENT_PATH $MERIDIO_TARGET_HELM_PATH
    cat $MERIDIO_TARGET_HELM_CURRENT_PATH/Chart.yaml | yq ".appVersion = \"$VERSION\" | .version = \"$HELM_CHART_VERSION\"" > $MERIDIO_TARGET_HELM_PATH/Chart.yaml
    cat $MERIDIO_TARGET_HELM_CURRENT_PATH/values.yaml | yq ".tag = \"$VERSION\" | .tapa.version = \"$VERSION\" | .exampleTarget.version = \"$VERSION\" | .registry = \"$REGISTRY\" | .repository = \"$REPOSITORY\"" > $MERIDIO_TARGET_HELM_PATH/values.yaml
    helm package $MERIDIO_TARGET_HELM_PATH --version $HELM_CHART_VERSION --destination $HELM_CHART_PATH
}

HELM_CHART_PATH="_output/helm/"
VERSION="${VERSION:-latest}"
HELM_CHART_VERSION="$VERSION"
REGISTRY="${REGISTRY:-registry.nordix.org}"
REPOSITORY="${REPOSITORY:-cloud-native/meridio}"
NSM_REPOSITORY="${NSM_REPOSITORY:-cloud-native/nsm}"

MERIDIO_TARGET_HELM_CURRENT_PATH="./examples/target/deployments/helm"

MERIDIO_HELM_PATH="$HELM_CHART_PATH/meridio"
MERIDIO_CRDS_HELM_PATH="$HELM_CHART_PATH/meridio-crds"
MERIDIO_TARGET_HELM_PATH="$HELM_CHART_PATH/meridio-target"

mkdir -p $HELM_CHART_PATH

# https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
# https://github.com/semver/semver/pull/724
if ! [[ $HELM_CHART_VERSION =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$ ]]; then
    HELM_CHART_VERSION="v0.0.0-$VERSION"
fi

chart_meridio
chart_meridio_crds
chart_meridio_target
