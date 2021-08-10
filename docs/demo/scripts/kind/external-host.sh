#! /bin/bash

parent_if_name="eth0"
vlan_if_name="vlan0"
vlan_id="100"
ip="169.254.100.150/24"
ip6="100:100::150/64"
trenches=(trench-a trench-b trench-c)

for index in {0..2}
do
    vi=$((vlan_id + index))
    container_name=${trenches[$index]}

    docker kill $container_name
    docker rm $container_name

    docker run -t -d --network="kind" --name="$container_name" --privileged registry.nordix.org/cloud-native/meridio/kind-host:latest

    docker exec -it $container_name sh -c "echo \"PS1='$container_name | vlan-id:$vi> '\" >> ~/.bashrc"

    docker exec -it $container_name sysctl -w net.ipv6.conf.all.disable_ipv6=0
    docker exec -it $container_name sysctl -w net.ipv4.fib_multipath_hash_policy=1
    docker exec -it $container_name sysctl -w net.ipv6.fib_multipath_hash_policy=1
    docker exec -it $container_name sysctl -w net.ipv6.conf.all.forwarding=1
    docker exec -it $container_name sysctl -w net.ipv4.conf.all.forwarding=1
    docker exec -it $container_name sysctl -w net.ipv6.conf.all.accept_dad=0

    docker exec -it $container_name ip link add link $parent_if_name name $vlan_if_name type vlan id $vi
    docker exec -it $container_name ip link set $vlan_if_name up
    docker exec -it $container_name ip addr add $ip dev $vlan_if_name
    docker exec -it $container_name ip addr add $ip6 dev $vlan_if_name

done
