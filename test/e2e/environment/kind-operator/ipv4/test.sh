#!/bin/bash

function init () {
    :
}

function end () {
    :
}

function on_failure() {
    OUTPUT_PATH="../../_output" OUTPUT_ID="$2" ../../hack/log_collector.sh
}

function configuration_new_vip () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/configuration-new-vip.yaml
}

function configuration_new_vip_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    kubectl delete vip -n red vip-a-2-v4
}

# Required to call the corresponding function
$1 $@
