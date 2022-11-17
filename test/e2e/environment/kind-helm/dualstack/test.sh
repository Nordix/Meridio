#!/bin/bash

function init () {
    kubectl patch configmap meridio-configuration-trench-a -n red --patch-file $(dirname -- $(readlink -fn -- "$0"))/configuration/init.yaml
}

function end () {
    :
}

function on_failure() {
    OUTPUT_PATH="../../_output" OUTPUT_ID="$2" ../../hack/log_collector.sh
}

function configuration_new_vip () {
    kubectl patch configmap meridio-configuration-trench-a -n red --patch-file $(dirname -- $(readlink -fn -- "$0"))/configuration/configuration-new-vip.yaml
}

function configuration_new_vip_revert () {
    kubectl patch configmap meridio-configuration-trench-a -n red --patch-file $(dirname -- $(readlink -fn -- "$0"))/configuration/init.yaml
}

# Required to call the corresponding function
$1 $@
