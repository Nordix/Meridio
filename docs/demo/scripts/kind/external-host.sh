#! /bin/bash

parent_if_name="eth0"
vlan_id="100"

trenches=(trench-a trench-b trench-c)
vlans=(vlan0 vlan1 vlan2)

for (( index_trench=0; index_trench<=$((${#trenches[@]}-1)); index_trench++ ))
do
    vi=$((vlan_id + index_trench * 100))
    container_name=${trenches[$index_trench]}

    docker kill $container_name || true
    docker rm $container_name || true

    docker run -t -d --network="kind" --name="$container_name" --privileged registry.nordix.org/cloud-native/meridio/kind-host:latest
    
    docker exec $container_name sh -c "echo \"PS1='[GW/TG] $container_name | VLAN:${vi}..$(($vi + ${#vlans[@]}-1))> '\" >> ~/.bashrc"

    docker exec $container_name sysctl -w net.ipv6.conf.all.disable_ipv6=0
    docker exec $container_name sysctl -w net.ipv4.fib_multipath_hash_policy=1
    docker exec $container_name sysctl -w net.ipv6.fib_multipath_hash_policy=1
    docker exec $container_name sysctl -w net.ipv6.conf.all.forwarding=1
    docker exec $container_name sysctl -w net.ipv4.conf.all.forwarding=1
    docker exec $container_name sysctl -w net.ipv6.conf.all.accept_dad=0

    for (( index_vlan=0; index_vlan<=$((${#vlans[@]}-1)); index_vlan++ ))
    do
        if_name=${vlans[$index_vlan]}
        v_id=$((vi + index_vlan))
        ip="169.254.$((vlan_id + index_vlan)).150/24"
        ip6="100:$((vlan_id + index_vlan))::150/64"
        
        docker exec $container_name ip link add link $parent_if_name name $if_name type vlan id $v_id
        docker exec $container_name ip link set $if_name up
        docker exec $container_name ip addr add $ip dev $if_name
        docker exec $container_name ip addr add $ip6 dev $if_name

        docker exec $container_name ethtool -K $parent_if_name tx off
    done

done
