#! /bin/bash

create_networks() {
    docker network create -d bridge kind --opt com.docker.network.driver.mtu=1500
    docker network create -d bridge external-bridge-network --subnet=fc00:100::0/64 --subnet=172.100.0.0/24 --opt com.docker.network.driver.mtu=1500 --ipv6
    docker network create -d bridge kind-bridge-network-1 --subnet=fc00:50::0/64 --subnet=172.50.0.0/24 --opt com.docker.network.driver.mtu=1300 --ipv6
    docker network create -d bridge kind-bridge-network-2 --subnet=fc00:51::0/64 --subnet=172.51.0.0/24 --opt com.docker.network.driver.mtu=1700 --ipv6
    kind_containers=$(docker ps -f name=kind -q)
    while IFS= read -r container; do
        docker network connect kind-bridge-network-1 $container
        docker network connect kind-bridge-network-2 $container
    done <<< "$kind_containers"
}

create_tg() {
    docker run -t -d --network="external-bridge-network" --name="$trench_name" --privileged registry.nordix.org/cloud-native/meridio/kind-host:latest
    docker exec $trench_name sh -c "echo \"PS1='[TG] $container_name> '\" >> ~/.bashrc"

    docker exec $trench_name sysctl -w net.ipv6.conf.all.disable_ipv6=0
    docker exec $trench_name sysctl -w net.ipv4.fib_multipath_hash_policy=1
    docker exec $trench_name sysctl -w net.ipv6.fib_multipath_hash_policy=1
    docker exec $trench_name sysctl -w net.ipv6.conf.all.forwarding=1
    docker exec $trench_name sysctl -w net.ipv4.conf.all.forwarding=1
    docker exec $trench_name sysctl -w net.ipv6.conf.all.accept_dad=0

    docker exec $container_name ethtool -K eth0 tx off
}

create_gw() {
    for gw_id in $gw_id_list
    do
        gw_name="$trench_name-gw-$gw_id"
        docker run -t -d --network="kind" --name="$gw_name" --privileged registry.nordix.org/cloud-native/meridio/kind-host:latest
        docker exec $gw_name sh -c "echo \"PS1='[GW] $gw_name> '\" >> ~/.bashrc"

        docker exec $gw_name sysctl -w net.ipv6.conf.all.disable_ipv6=0
        docker exec $gw_name sysctl -w net.ipv4.fib_multipath_hash_policy=1
        docker exec $gw_name sysctl -w net.ipv6.fib_multipath_hash_policy=1
        docker exec $gw_name sysctl -w net.ipv6.conf.all.forwarding=1
        docker exec $gw_name sysctl -w net.ipv4.conf.all.forwarding=1
        docker exec $gw_name sysctl -w net.ipv6.conf.all.accept_dad=0

        docker network connect kind-bridge-network-1 $gw_name
        docker network connect kind-bridge-network-2 $gw_name
        docker network connect external-bridge-network $gw_name

        docker exec $gw_name ip link add link eth0 name vlan0 type vlan id 100
        docker exec $gw_name ip link set vlan0 up
        docker exec $gw_name ip addr add 169.254.100.15$gw_id/24 dev vlan0
        docker exec $gw_name ip addr add 100:100::15$gw_id/64 dev vlan0

        docker exec $gw_name ip link add link eth0 name vlan1 type vlan id 101
        docker exec $gw_name ip link set vlan1 up
        docker exec $gw_name ip addr add 169.254.101.15$gw_id/24 dev vlan1
        docker exec $gw_name ip addr add 100:101::15$gw_id/64 dev vlan1

        docker exec $gw_name ip link add link eth0 name vlan2 type vlan id 102
        docker exec $gw_name ip link set vlan2 up
        docker exec $gw_name ip addr add 169.254.102.15$gw_id/24 dev vlan2
        docker exec $gw_name ip addr add 100:102::15$gw_id/64 dev vlan2

        docker exec $gw_name ip link add link eth0 name vlan3 type vlan id 103
        docker exec $gw_name ip link set vlan3 up
        docker exec $gw_name ip addr add 169.254.103.15$gw_id/24 dev vlan3
        docker exec $gw_name ip addr add 100:103::15$gw_id/64 dev vlan3

        docker exec $gw_name ip link add link eth1 name vlan10 type vlan id 110
        docker exec $gw_name ip link set vlan10 up
        docker exec $gw_name ip addr add 169.254.110.15$gw_id/24 dev vlan10
        docker exec $gw_name ip addr add 100:110::15$gw_id/64 dev vlan10

        docker exec $gw_name ip link add link eth2 name vlan20 type vlan id 120
        docker exec $gw_name ip link set vlan20 up
        docker exec $gw_name ip addr add 169.254.120.15$gw_id/24 dev vlan20
        docker exec $gw_name ip addr add 100:120::15$gw_id/64 dev vlan20

        docker exec $gw_name ethtool -K eth0 tx off
        docker exec $gw_name ethtool -K eth1 tx off
        docker exec $gw_name ethtool -K eth2 tx off
        docker exec $gw_name ethtool -K eth3 tx off
    done
}

# remove all containers GW and TG and remove all networks
clear() {
    for trench in $TRENCH_LIST
    do
        trench_name="trench-$trench"
        docker kill trench-$trench || true
        docker rm trench-$trench || true
        for gw_id in $gw_id_list
        do
            gw_name="$trench_name-gw-$gw_id"
            docker kill $gw_name || true
            docker rm $gw_name || true
        done
    done
    docker network rm kind
    docker network rm external-bridge-network
    docker network rm kind-bridge-network-1
    docker network rm kind-bridge-network-2
}

all() {
    clear
    # create_networks
    # for trench in $TRENCH_LIST
    # do
    #     trench_name="trench-$trench"
    #     create_tg
    #     create_gw
    # done
}

gw_id_list="0 1"
TRENCH_LIST=${TRENCH_LIST:-"a b"}

all
