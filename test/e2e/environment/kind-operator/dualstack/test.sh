#!/bin/bash

function init () {
    :
}

function end () {
    :
}

function on_failure() {
    OUTPUT_PATH="../../_output" OUTPUT_ID="$2" EXEC_NAMESPACE="red" EXEC_STATELESS_LB_FRONTEND_LABELS="app=stateless-lb-frontend-attractor-a-1 app=stateless-lb-frontend-attractor-b-1" EXEC_PROXY_LABELS="app=proxy-conduit-a-1 app=proxy-conduit-b-1" EXEC_TARGETS_LABELS="app=target-a app=target-b" ../../hack/log_collector.sh
}

function new_vip () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-vip.yaml
}

function new_vip_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    kubectl delete vip -n red vip-a-2-v4
    kubectl delete vip -n red vip-a-2-v6
}

function delete_create_trench () {
    kubectl delete trench trench-a -n red
    sleep 5
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
    # Wait for all pods to be in running state (no Terminating pods)
    while kubectl get pods -n red --no-headers | awk '$3' | grep -v "Running" > /dev/null; do sleep 1; done
}

function delete_create_trench_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    sleep 5
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
}

function new_stream () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-stream.yaml
}

function new_stream_revert () {
    kubectl delete stream -n red stream-a-iii
    kubectl delete flow -n red flow-a-x-tcp
}

function stream_max_targets () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/stream-max-targets.yaml
}

function stream_max_targets_revert () {
    kubectl delete stream -n red stream-a-iii
    kubectl delete flow -n red flow-a-x-tcp
}

# Required to call the corresponding function
$1 $@
