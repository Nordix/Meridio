#!/bin/bash

if ! command -v yq &> /dev/null
then
    echo "yq binary not found, installing... "
    snap install yq
fi


ENABLE_MUTATING_WEBHOOK="${ENABLE_MUTATING_WEBHOOK:-true}"

if ! ${ENABLE_MUTATING_WEBHOOK}; then
    yq -iy 'del(.resources[] | select(. == "../webhook-configuration-mutating"))' config/default/kustomization.yaml
fi