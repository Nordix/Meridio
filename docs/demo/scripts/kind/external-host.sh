#! /bin/bash

while echo "$1" | grep -q '^--'; do
        if echo $1 | grep -q =; then
                o=$(echo "$1" | cut -d= -f1 | sed -e 's,-,_,g')
                v=$(echo "$1" | cut -d= -f2-)
                eval "$o=\"$v\""
        else
                o=$(echo "$1" | sed -e 's,-,_,g')
                eval "$o=yes"
        fi
        shift
done
unset o v
long_opts=`set | grep '^__' | cut -d= -f1`

test -n "$__default_route" || __default_route=yes  # BGP to annouce default routes by default
test -n "$__network_name" || export __network_name=kind
parent_if_name="eth0"
vlan_id="100"

trenches=(trench-a trench-b trench-c)
vlans=(vlan0 vlan1 vlan2)

for (( index_trench=0; index_trench<=$((${#trenches[@]}-1)); index_trench++ ))
do
    vi=$((vlan_id + index_trench * 100))
    container_name=${trenches[$index_trench]}
    cmd=""

    docker kill $container_name || true
    docker rm $container_name || true

    if [ "$__default_route" == "yes" ]; then
        docker run -t -d --network=$__network_name --name="$container_name" --privileged registry.nordix.org/cloud-native/meridio/kind-host:latest
    else
        docker run -t -d --network=$__network_name --name="$container_name" --privileged registry.nordix.org/cloud-native/meridio/kind-host:latest \
            /bin/sh -c "sleep 5 ; /usr/sbin/bird -d -c /etc/bird/bird-gw-no-default.conf"
    fi
    
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

        # Create IP addresses for traffic tests that are not part of the network shared with
        # external interfaces on the connected cluster. Thus, cluster side device routes wouldn't
        # interfere.
        # Note: local BGP must announce these networks
        tip="200.100.0.$((vlan_id + index_vlan))/32"
        tip6="200:100::$((vlan_id + index_vlan))/128"
        
        docker exec $container_name ip link add link $parent_if_name name $if_name type vlan id $v_id
        docker exec $container_name ip link set $if_name up
        docker exec $container_name ip addr add $ip dev $if_name
        docker exec $container_name ip addr add $ip6 dev $if_name
        docker exec $container_name ip addr add $tip dev $if_name
        docker exec $container_name ip addr add $tip6 dev $if_name

        docker exec $container_name ethtool -K $parent_if_name tx off
    done

done
