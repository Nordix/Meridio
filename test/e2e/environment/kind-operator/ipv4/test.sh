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

function new_flow () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-flow.yaml
}

function new_flow_revert () {
    kubectl delete flow -n red flow-a-x-tcp
}

function flow_priority () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/flow-priority.yaml
}

function flow_priority_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
}

# Required to call the corresponding function
$1 $@
