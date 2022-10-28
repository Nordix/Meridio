#!/bin/bash

if ! command -v yq &> /dev/null
then
    echo "yq binary not found, installing... "
    snap install yq
fi


ENABLE_MUTATING_WEBHOOK="${ENABLE_MUTATING_WEBHOOK:-true}"
WEBHOOK_SUPPORT="${WEBHOOK_SUPPORT:-spire}"

if [[ ${WEBHOOK_SUPPORT} != "spire" && ${WEBHOOK_SUPPORT} != "certmanager" ]]; then
    echo "unknown WEBHOOK_SUPPORT, please use 'spire' or 'certmanager'"
    exit 1
fi

spire_kustomization="config/spire/kustomization.yaml"
cert_kustomization="config/certmanager/kustomization.yaml"

case ${WEBHOOK_SUPPORT} in
    "spire")
        kustomization=$spire_kustomization
    ;;
    "certmanager")
        kustomization=$cert_kustomization
    ;;
    *)
    echo "unknown"
    ;;
esac

# if ENABLE_MUTATING_WEBHOOK is not true, remove the relevant references in kustomization files to not generate mutating webhook configuration manifest
if [[ ${ENABLE_MUTATING_WEBHOOK} != true ]] ; then
    # remove "../webhook-configuration-mutating" from resources field in kustomization.yaml, and corresponding CA injection label/annotation
    yq e -i 'del(.resources[] | select(. == "../webhook-configuration-mutating"))' "$kustomization"
    yq e -i 'del(.patchesStrategicMerge[] | select(. == "patches/mwebhookcainjection_patch.yaml"))' "$kustomization"
else
    # add "../webhook-configuration-mutating" from resources field in kustomization.yaml, and corresponding CA injection label/annotation
    if [[ $(yq 'contains({"resources": ["../webhook-configuration-mutating"]})' ${kustomization}) == false ]]; then
        yq e -i '.resources += ["../webhook-configuration-mutating"]' "${kustomization}"
    fi
    if [[ $(yq 'contains({"patchesStrategicMerge": ["patches/mwebhookcainjection_patch.yaml"]})' ${kustomization}) == false ]]; then
        yq e -i '.patchesStrategicMerge += ["patches/mwebhookcainjection_patch.yaml"]' "$kustomization"
    fi
fi