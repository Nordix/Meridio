#!/bin/bash

function init () {
    :
}

function end () {
    :
}

function configuration_new_vip () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/configuration-new-vip.yaml
}

function configuration_new_vip_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    kubectl delete vip -n red vip-a-2-v6
}

# Required to call the corresponding function
$1 $@
