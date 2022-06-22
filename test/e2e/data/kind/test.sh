#!/bin/bash

function init () {
    kubectl patch configmap meridio-configuration-trench-a -n red --patch-file $(dirname -- $(readlink -fn -- "$0"))/init.yaml
}

function end () {
    :
}

function configuration_new_ip () {
    kubectl patch configmap meridio-configuration-trench-a -n red --patch-file $(dirname -- $(readlink -fn -- "$0"))/configuration-new-vip.yaml
}

function configuration_new_ip_revert () {
    kubectl patch configmap meridio-configuration-trench-a -n red --patch-file $(dirname -- $(readlink -fn -- "$0"))/init.yaml
}

# Required to call the corresponding function
$1 $@