#!/bin/bash

function init {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/init.yaml
}

function configuration_new_ip () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration-new-vip.yaml
}

# Required to call the corresponding function
$1 $@
