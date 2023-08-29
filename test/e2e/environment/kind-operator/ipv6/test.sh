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
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    kubectl delete vip -n red vip-a-2-v6
}

function delete_create_trench () {
    kubectl delete trench trench-a -n red
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    sleep 10
    # Wait for all pods to be in running state (no Terminating pods)
    while kubectl get pods -n red --no-headers | awk '$3' | grep -v "Running" > /dev/null; do sleep 1; done
}

function delete_create_trench_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    sleep 10
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
}

function new_stream () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-stream.yaml
}

function new_stream_revert () {
    kubectl delete stream -n red stream-a-iii
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    kubectl delete flow -n red flow-a-x-tcp
}

function stream_max_targets () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/stream-max-targets.yaml
}

function stream_max_targets_revert () {
    kubectl delete stream -n red stream-a-iii
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    kubectl delete flow -n red flow-a-x-tcp
}

function new_flow () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-flow.yaml
    sleep 5
}

function new_flow_revert () {
    kubectl delete flow -n red flow-a-x-tcp
    sleep 5
}

function flow_priority () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/flow-priority.yaml
    sleep 5
}

function flow_priority_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    sleep 5
}

function flow_destination_ports_range () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/flow-destination-ports-range.yaml
    sleep 5
}

function flow_destination_ports_range_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    sleep 5
}

function flow_byte_matches () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/flow-byte-matches.yaml
    sleep 5
}

function flow_byte_matches_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    sleep 5
}

function new_attractor_nsm_vlan () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-attractor-nsm-vlan.yaml
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    sleep 5
    kubectl wait --for=condition=Ready pods --all -n red --timeout=4m
}

function new_attractor_nsm_vlan_revert () {
    kubectl delete -f $(dirname -- $(readlink -fn -- "$0"))/configuration/new-attractor-nsm-vlan.yaml
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    sleep 5
    while kubectl get pods -n red --no-headers | awk '$3' | grep -v "Running" > /dev/null; do sleep 1; done
}

function conduit_destination_port_nats () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/conduit-destination-port-nats.yaml
    sleep 5
}

function conduit_destination_port_nats_revert () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/init-trench-a.yaml
    sleep 5
}

function kill_ipam () {
    litmus "kill-ipam"
}

function kill_nsp () {
    litmus "kill-nsp"
}

function kill_operator () {
    litmus "kill-operator"
}

function kill_proxy () {
    litmus "kill-proxy"
}

function kill_stateless_lb () {
    litmus "kill-stateless-lb"
}

function kill_frontend () {
    litmus "kill-frontend"
}

function litmus () {
    kubectl apply -f $(dirname -- $(readlink -fn -- "$0"))/configuration/resiliency-$1.yaml
    if [ ! $? -eq 0 ]; then exit 1 ; fi
    sleep 5
    # Wait for the restart
    while [ "$(kubectl get ChaosEngine container-$1 -n litmus -o json | jq -r '.status.engineStatus')" != "completed" ] ; do sleep 1; done
    # Checks the verdict
    if [[ "$(kubectl get ChaosEngine container-$1 -n litmus -o json | jq -r '.status.experiments[] | select(.verdict != "Pass") | [.verdict] | @tsv' 2>&1 | wc -l)" != "0" ]]; then
        >&2 echo "$1 failed"
        exit 1
    fi
    kubectl delete -f $(dirname -- $(readlink -fn -- "$0"))/configuration/resiliency-$1.yaml
}

# Required to call the corresponding function
$1 $@
