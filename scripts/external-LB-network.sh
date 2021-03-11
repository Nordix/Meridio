#! /bin/bash

parent_if_name="eth0"
vlan_if_name="vlan0"
vlan_id="68"
bridge_if_name="vlan0-bridge"
app_name="load-balancer"

# add vlan and bridge on host
set_host_bridges () {
    echo "----------------";
    echo "add vlan and bridge on host";
    all_hosts=($(kubectl get pods --no-headers -o wide --all-namespaces | awk '{print $8}' | sort | uniq))
    for index in ${!all_hosts[@]}
    do
        host=${all_hosts[index]}
        echo "Setting bridge: $host";

        id=$((index + 100))
        docker exec -it $host ip link add link $parent_if_name name $vlan_if_name type vlan id $vlan_id
        docker exec -it $host ip link add $bridge_if_name type bridge
        docker exec -it $host ip addr add 172.10.10.$id/24 dev $bridge_if_name
        docker exec -it $host ip link set $vlan_if_name master $bridge_if_name
        docker exec -it $host ip link set $vlan_if_name up
        docker exec -it $host ip link set $bridge_if_name up
    done
}

# attach pods to the bridge with veth
attach_load_balancers () {
    echo "----------------";
    echo "attach pods to the bridge with veth";

    pods=($(kubectl get pods --selector=app=$app_name --no-headers | awk '{print $1}'))
    hosts=($(kubectl get pods --selector=app=$app_name --no-headers -o wide | awk '{print $7}'))
    for index in ${!pods[@]}
    do
        pod=${pods[index]}
        host=${hosts[index]}
        echo "Setting link: $pod - $host";

        if $(kubectl exec $pod -- ip a | grep -q "172.10.10")
        then 
            echo "Link already exists";
        else
            hash=$(echo $pod | tail -c 5)
            pod_ns_name=$(docker exec -it $host bash -c "grep $pod /proc/[0-9]*/environ | head -1 | cut -d '/' -f3" | sed 's/[^0-9]*//g')
            veth_pod_if_name="ve-pod-$hash"
            veth_bridge_if_name="ve-br-$hash"

            echo "ns: $pod_ns_name ; pod if name: $veth_pod_if_name ; bridge veth: $veth_bridge_if_name"

            docker exec -it $host ip link add $veth_bridge_if_name type veth peer name $veth_pod_if_name
            docker exec -it $host ip link set $veth_bridge_if_name master $bridge_if_name
            docker exec -it $host ip link set $veth_bridge_if_name up

            docker exec -it $host ip link set $veth_pod_if_name netns $pod_ns_name
            id=$((index + 1))
            echo "ip: 172.10.10.$id/24"
            kubectl exec $pod -- ip link set $veth_pod_if_name up
            kubectl exec $pod -- ip addr add 172.10.10.$id/24 dev $veth_pod_if_name
        fi
    done
}

set_host_bridges
attach_load_balancers
