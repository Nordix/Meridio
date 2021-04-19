#! /bin/bash

parent_if_name="eth0"
vlan_if_name="vxlan-ext"
vlan_id="1500"
app_name="load-balancer"

attach_load_balancers () {
    echo "----------------";
    echo "attach new vxlan to the pods";

    pods=($(kubectl get pods --sort-by=.metadata.creationTimestamp --selector=app=$app_name --no-headers | awk '{print $1}'))
    hosts=($(kubectl get pods --sort-by=.metadata.creationTimestamp --selector=app=$app_name --no-headers -o wide | awk '{print $7}'))
    for index in ${!pods[@]}
    do
        pod=${pods[index]}
        host=${hosts[index]}
        echo "Setting link: $pod - $host";

        if $(kubectl exec $pod -- ip a | grep -q "192.168.")
        then 
            echo "Link already exists";
        else
            hash=$(echo $pod | tail -c 5)
            pod_ns_name=$(docker exec -it $host bash -c "grep $pod /proc/[0-9]*/environ | head -1 | cut -d '/' -f3" | sed 's/[^0-9]*//g')

            id=$((index + 1))
            vni=$((vlan_id + id))
            if_name="${vlan_if_name}-${id}"
            ip="192.168.${id}.1/24"
            ip6="1500:${id}::1/64"

            echo "ns: $pod_ns_name ; pod if name: $if_name ; vni: $vni ; ip: $ip ; ipv6: $ip6"

            docker exec -it $host ip link add $if_name type vxlan id $vni group 239.1.$id.1 dev $parent_if_name dstport 4789
            docker exec -it $host ip link set $if_name netns $pod_ns_name
            kubectl exec $pod -c load-balancer -- ip link set $if_name up
            kubectl exec $pod -c load-balancer -- ip addr add $ip dev $if_name
            kubectl exec $pod -c load-balancer -- ip addr add $ip6 dev $if_name
            kubectl exec $pod -c load-balancer -- ip route add 192.168.0.0/16 dev $if_name
            kubectl exec $pod -c load-balancer -- ip -6 route add 1500::0/16 dev $if_name
        fi
    done
}

attach_load_balancers