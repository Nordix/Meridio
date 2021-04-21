#! /bin/bash

parent_if_name="eth0"
vlan_if_name="vxlan-ext"
vlan_id="1500"
app_name="load-balancer"

attach_load_balancers () {
    echo "----------------";
    echo "attach new vxlan to the pods";

    pods=($(kubectl get pods --sort-by=.metadata.creationTimestamp --selector=app=$app_name --no-headers -o wide --all-namespaces | awk '{print $2}'))
    hosts=($(kubectl get pods --sort-by=.metadata.creationTimestamp --selector=app=$app_name --no-headers -o wide --all-namespaces | awk '{print $8}'))
    states=($(kubectl get pods --sort-by=.metadata.creationTimestamp --selector=app=$app_name --no-headers -o wide --all-namespaces | awk '{print $4}'))
    namespaces=($(kubectl get pods --sort-by=.metadata.creationTimestamp --selector=app=$app_name --no-headers -o wide --all-namespaces | awk '{print $1}'))
    for index in ${!pods[@]}
    do
        pod=${pods[index]}
        host=${hosts[index]}
        state=${states[index]}
        namespace=${namespaces[index]}

        if [ $state != "Running" ]
        then
            continue
        fi

        if $(kubectl exec $pod -c load-balancer -n $namespace -- ip a | grep -q "192.168.")
        then 
            echo "Link already exists";
        else
            pod_ns_name=$(docker exec -it $host bash -c "grep $pod /proc/[0-9]*/environ | head -1 | cut -d '/' -f3" | sed 's/[^0-9]*//g')

            id=$((index + 1))
            vni=$((vlan_id + id))
            if_name="${vlan_if_name}-${id}"
            ip="192.168.${id}.1/24"
            ip6="1500:${id}::1/64"

            echo "pod: $pod ($namespace) ; host: $host ; ns: $pod_ns_name ; pod if name: $if_name ; vni: $vni ; ip: $ip ; ipv6: $ip6"

            docker exec -it $host ip link add $if_name type vxlan id $vni group 239.1.$id.1 dev $parent_if_name dstport 4789
            docker exec -it $host ip link set $if_name netns $pod_ns_name
            kubectl exec $pod -c load-balancer -n $namespace -- ip link set $if_name up
            kubectl exec $pod -c load-balancer -n $namespace -- ip addr add $ip dev $if_name
            kubectl exec $pod -c load-balancer -n $namespace -- ip addr add $ip6 dev $if_name
            kubectl exec $pod -c load-balancer -n $namespace -- ip route add 192.168.0.0/16 dev $if_name
            kubectl exec $pod -c load-balancer -n $namespace -- ip -6 route add 1500::0/16 dev $if_name
        fi
    done
}

attach_load_balancers