#!/bin/bash

function init () {
    :
}

function end () {
    :
}

function on_failure() {
    OUTPUT_PATH="../../_output" OUTPUT_ID="$2" EXEC_NAMESPACE="red" EXEC_STATELESS_LB_FRONTEND_LABELS="app=stateless-lb-frontend-attractor-a-1 app=stateless-lb-frontend-attractor-b-1 app=stateless-lb-frontend-attractor-a-2 app=stateless-lb-frontend-attractor-a-3" EXEC_PROXY_LABELS="app=proxy-conduit-a-1 app=proxy-conduit-b-1 app=proxy-conduit-a-2 app=proxy-conduit-a-3" EXEC_TARGETS_LABELS="app=target-a app=target-b" ../../hack/log_collector.sh
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
    sleep 10
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
    # Wait for all pods to be in running state (no Terminating pods)
    while kubectl get pods -n red --no-headers | awk '$3' | grep -v "Running" > /dev/null; do sleep 1; done
}

function delete_create_trench_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    sleep 10
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

function flow_destination_ports_range () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/flow-destination-ports-range.yaml
}

function flow_destination_ports_range_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
}

function flow_byte_matches () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/flow-byte-matches.yaml
}

function flow_byte_matches_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
}

function new_attractor_nsm_vlan () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-attractor-nsm-vlan.yaml
    sleep 10
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
}

function new_attractor_nsm_vlan_revert () {
    kubectl delete -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-attractor-nsm-vlan.yaml
    sleep 10
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
}

function conduit_destination_port_nats () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/conduit-destination-port-nats.yaml
}

function conduit_destination_port_nats_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
}

# Required to call the corresponding function
$1 $@
