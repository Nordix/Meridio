#! /bin/bash

collect_namespaced_resource() {
    mkdir -p "$full_output_path/$1/describe"
    mkdir -p "$full_output_path/$1/yaml"
    kubectl describe $1 $3 -n $2 > "$full_output_path/$1/describe/$2.$3.txt" 2>/dev/null
    kubectl get $1 $3 -n $2 -o yaml > "$full_output_path/$1/yaml/$2.$3.yaml" 2>/dev/null
}

collect_not_namespaced_resource() {
    mkdir -p "$full_output_path/$1/describe"
    mkdir -p "$full_output_path/$1/yaml"
    kubectl describe $1 $2 > "$full_output_path/$1/describe/$2.txt" 2>/dev/null
    kubectl get $1 $2 -o yaml > "$full_output_path/$1/yaml/$2.yaml" 2>/dev/null
}

collect_resource() {
    resources=$(kubectl get $1 -o wide --all-namespaces 2>/dev/null)
    echo "$resources" > "$full_output_path/$1/all.txt"
    resources_no_header=$(echo "$resources" | sed '1d')
    while IFS= read -r resource; do
        if [ -z "$resource" ]; then
            continue
        fi
        if [[ "$2" == "true" ]]; then
            name=$(echo "$resource" | awk '{print $2}')
            namespace=$(echo "$resource" | awk '{print $1}')
            collect_namespaced_resource $1 $namespace $name
        else
            name=$(echo "$resource" | awk '{print $1}')
            collect_not_namespaced_resource $1 $name
        fi
    done <<< "$resources_no_header"
}

collect_top() {
    echo "collecting top ..."
    kubectl top pods --all-namespaces >> "$full_output_path/top.txt" 2>/dev/null
}

collect_logs() {
    pods=$(kubectl get pods --all-namespaces --no-headers=true)
    while IFS= read -r pod; do
        name=$(echo "$pod" | awk '{print $2}')
        namespace=$(echo "$pod" | awk '{print $1}')
        containers=$(kubectl get pods $name -n $namespace -o jsonpath="{.spec.containers[*].name}")
        init_containers=$(kubectl get pods $name -n $namespace -o jsonpath="{.spec.initContainers[*].name}")
        echo "collecting logs of $name.$namespace ..."
        mkdir -p "$full_output_path/pods/logs"
        mkdir -p "$full_output_path/pods/logs/previous"
        for container in $containers; do
            kubectl logs $name -n $namespace -c $container > "$full_output_path/pods/logs/$namespace.$name.$container.log"
            kubectl logs $name -n $namespace -c $container --previous=true > "$full_output_path/pods/logs/previous/$namespace.$name.$container.log" 2>/dev/null
        done
        for container in $init_containers; do
            kubectl logs $name -n $namespace -c $container > "$full_output_path/pods/logs/$namespace.$name.$container.log"
            kubectl logs $name -n $namespace -c $container --previous=true > "$full_output_path/pods/logs/previous/$namespace.$name.$container.log" 2>/dev/null
        done
    done <<< "$pods"
}

collect_exec_vpp_forwarder() {
    mkdir -p "$full_output_path/pods"
    mkdir -p "$full_output_path/pods/exec"
    pods=$(kubectl get pods -n $EXEC_NSM_NAMESPACE --no-headers=true --selector="$EXEC_NSM_FORWARDER_LABEL")
    pod_names=$(echo "$pods" | awk '{print $1}')
    for pod_name in $pod_names
    do
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- ip a > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.ip-a.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show interface address > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-interface-address.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show interface > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-interface.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show tap > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-tap.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show mode > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-mode.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show bridge-domain > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-bridge-domain.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show vxlan tunnel raw > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-vxlan-tunnel-raw.txt" 2>/dev/null
        kubectl exec $pod_name -n $EXEC_NSM_NAMESPACE -- vppctl show acl-plugin acl > "$full_output_path/pods/exec/$EXEC_NSM_NAMESPACE.$pod_name.vppctl-show-acl-plugin-acl.txt" 2>/dev/null
    done
}

collect_exec_stateless_lb_frontend() {
    mkdir -p "$full_output_path/pods"
    mkdir -p "$full_output_path/pods/exec"
    for label in $EXEC_STATELESS_LB_FRONTEND_LABELS
    do
        pods=$(kubectl get pods -n $EXEC_NAMESPACE --no-headers=true --selector="$label")
        pod_names=$(echo "$pods" | awk '{print $1}')
        for pod_name in $pod_names
        do
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- ip a > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-a.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- ip rule > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-rule.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- ip -6 rule > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-6-rule.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- ip route show table all > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-route-show-table-all.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- nft list ruleset > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.nft-list-ruleset.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c stateless-lb -- ps aux > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ps-aux.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c stateless-lb -- nfqlb flow-list > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.nfqlb-flow-list.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- netstat -s > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.netstat-s.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- netstat -6 -s > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.netstat-6-s.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- ip neighbour > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-neighbour.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c frontend -- birdc -s /var/run/bird/bird.ctl show protocol all > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.birdc-show-protocol-all.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c frontend -- cat /var/log/bird.log > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.bird.log" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_STATELESS_LB_CONTAINER -- cat /proc/net/dev > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.proc-net-dev.txt" 2>/dev/null
            shared_memory=$(kubectl exec $pod_name -n $EXEC_NAMESPACE -c stateless-lb -- ls /dev/shm/ | grep "tshm-")
            while IFS= read -r shm; do
                kubectl exec $pod_name -n $EXEC_NAMESPACE -c stateless-lb -- nfqlb show --shm=$shm > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.nfqlb-show-shm-$shm.txt" 2>/dev/null
            done <<< "$shared_memory"
        done
    done
}

collect_exec_proxy() {
    mkdir -p "$full_output_path/pods"
    mkdir -p "$full_output_path/pods/exec"
    for label in $EXEC_PROXY_LABELS
    do
        pods=$(kubectl get pods -n $EXEC_NAMESPACE --no-headers=true --selector="$label")
        pod_names=$(echo "$pods" | awk '{print $1}')
        for pod_name in $pod_names
        do
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- ip a > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-a.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- ip rule > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-rule.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- ip -6 rule > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-6-rule.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- ip route show table all > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-route-show-table-all.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- ps aux > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ps-aux.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- netstat -s > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.netstat-s.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- netstat -6 -s > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.netstat-6-s.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- ip neighbour > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-neighbour.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- bridge fdb > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.bridge-fdb.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -- cat /proc/net/dev > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.proc-net-dev.txt" 2>/dev/null
        done
    done
}

collect_exec_targets() {
    mkdir -p "$full_output_path/pods"
    mkdir -p "$full_output_path/pods/exec"
    for label in $EXEC_TARGETS_LABELS
    do
        pods=$(kubectl get pods -n $EXEC_NAMESPACE --no-headers=true --selector="$label")
        pod_names=$(echo "$pods" | awk '{print $1}')
        for pod_name in $pod_names
        do
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- ip a > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-a.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- ip rule > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-rule.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- ip -6 rule > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-6-rule.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- ip route show table all > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-route-show-table-all.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- ps aux > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ps-aux.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- timeout --preserve-status 0.5 ./target-client watch > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.target-client-watch.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- netstat -s > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.netstat-s.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- netstat -6 -s > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.netstat-6-s.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- ip neighbour > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.ip-neighbour.txt" 2>/dev/null
            kubectl exec $pod_name -n $EXEC_NAMESPACE -c $EXEC_TARGET_CONTAINER -- cat /proc/net/dev > "$full_output_path/pods/exec/$EXEC_NAMESPACE.$pod_name.proc-net-dev.txt" 2>/dev/null
        done
    done
}

collect_all() {
    resources=$(kubectl api-resources --verbs=get)
    kubectl api-resources -o wide > "$full_output_path/api-resources.txt"
    kubectl get pods --all-namespaces -o wide > "$full_output_path/get-pods.txt"
    resources_no_header=$(echo "$resources" | sed '1d')
    collect_top
    if [[ "$COLLECT_EXECS" == "true" ]]; then
        collect_exec_stateless_lb_frontend
        collect_exec_proxy
        collect_exec_targets
        collect_exec_vpp_forwarder
    fi
    if [[ "$COLLECT_RESOURCES" == "true" ]]; then
        while IFS= read -r resource; do
            namespaced=$(echo "$resource" | awk '{print $(NF-1)}')
            resource_name=$(echo "$resource" | awk '{print $1}')
            if [[ " ${EXCLUDE_RESOURCES[*]} " =~ " ${resource_name} " ]]; then
                continue
            fi
            mkdir -p "$full_output_path/$resource_name"
            echo "collecting $resource_name ..."
            collect_resource $resource_name $namespaced
        done <<< "$resources_no_header"
    fi
    if [[ "$COLLECT_LOGS" == "true" ]]; then
        collect_logs
    fi
}

timestamp=$(date +%s)

EXCLUDE_RESOURCES="bindings componentstatuses events limitranges podtemplates replicationcontrollers resourcequotas controllerrevisions tokenreviews localsubjectaccessreviews selfsubjectaccessreviews selfsubjectrulesreviews subjectaccessreviews certificatesigningrequests leases events flowschemas prioritylevelconfigurations runtimeclasses priorityclasses apiservices csinodes csistoragecapacities"

EXEC_TARGET_CONTAINER=${EXEC_TARGET_CONTAINER:-"example-target"}
EXEC_STATELESS_LB_CONTAINER=${EXEC_STATELESS_LB_CONTAINER:-"stateless-lb"}

COLLECT_LOGS=${COLLECT_LOGS:-"true"}
COLLECT_RESOURCES=${COLLECT_RESOURCES:-"true"}
COLLECT_EXECS=${COLLECT_EXECS:-"true"}

EXEC_NSM_NAMESPACE=${EXEC_NSM_NAMESPACE:-"nsm"}
EXEC_NSM_FORWARDER_LABEL=${EXEC_NSM_FORWARDER_LABEL:-"app=forwarder-vpp"}

EXEC_NAMESPACE=${EXEC_NAMESPACE:-""}
EXEC_STATELESS_LB_FRONTEND_LABELS=${EXEC_STATELESS_LB_FRONTEND_LABELS:-""}
EXEC_PROXY_LABELS=${EXEC_PROXY_LABELS:-""}
EXEC_TARGETS_LABELS=${EXEC_TARGETS_LABELS:-""}

OUTPUT_ID=${OUTPUT_ID:-$timestamp}
OUTPUT_PATH=${OUTPUT_PATH:-"_output"}
collector_output_path=$OUTPUT_PATH"/log_collector"
full_output_path=$collector_output_path"/"$OUTPUT_ID

echo $OUTPUT_ID
echo $OUTPUT_PATH
echo $collector_output_path
echo $full_output_path

rm -rf $full_output_path
mkdir -p $OUTPUT_PATH
mkdir -p $collector_output_path
mkdir -p $full_output_path

collect_all

rm -rf $OUTPUT_PATH/log_collector_$OUTPUT_ID.tgz
tar -cvzf $OUTPUT_PATH/log_collector_$OUTPUT_ID.tgz $full_output_path > /dev/null 2>&1
rm -rf $full_output_path

echo "-----------------------"
echo "Log file available: $OUTPUT_PATH/log_collector_$OUTPUT_ID.tgz"
echo "-----------------------"

# Example: EXEC_NAMESPACE="red" EXEC_STATELESS_LB_FRONTEND_LABELS="app=stateless-lb-frontend-attractor-a-1 app=stateless-lb-frontend-attractor-b-1 app=stateless-lb-frontend-attractor-a-2 app=stateless-lb-frontend-attractor-a-3" EXEC_PROXY_LABELS="app=proxy-conduit-a-1 app=proxy-conduit-b-1 app=proxy-conduit-a-2 app=proxy-conduit-a-3" EXEC_TARGETS_LABELS="app=target-a app=target-b" ./hack/log_collector.sh

# Example 2: COLLECT_LOGS="false" COLLECT_RESOURCES="false" EXEC_NAMESPACE="red" EXEC_STATELESS_LB_FRONTEND_LABELS="app=stateless-lb-frontend-attractor-a-1 app=stateless-lb-frontend-attractor-b-1 app=stateless-lb-frontend-attractor-a-2 app=stateless-lb-frontend-attractor-a-3" EXEC_PROXY_LABELS="app=proxy-conduit-a-1 app=proxy-conduit-b-1 app=proxy-conduit-a-2 app=proxy-conduit-a-3" EXEC_TARGETS_LABELS="app=target-a app=target-b" ./hack/log_collector.sh
