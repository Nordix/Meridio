#! /bin/bash

docker kill ubuntu-ext
docker rm ubuntu-ext

docker run -t -d --network="kind" --name="ubuntu-ext" --privileged ubuntu

docker exec -it ubuntu-ext apt-get update -y --fix-missing
docker exec -it ubuntu-ext apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat wget

docker exec -it ubuntu-ext wget https://github.com/Nordix/ctraffic/releases/download/v1.3.0/ctraffic.gz
docker exec -it ubuntu-ext gunzip ctraffic.gz 
docker exec -it ubuntu-ext chmod u+x ctraffic
docker exec -it ubuntu-ext mv ctraffic /usr/bin/

docker exec -it ubuntu-ext sysctl -w net.ipv6.conf.all.disable_ipv6=0
docker exec -it ubuntu-ext sysctl -w net.ipv4.fib_multipath_hash_policy=1
docker exec -it ubuntu-ext sysctl -w net.ipv6.fib_multipath_hash_policy=1
docker exec -it ubuntu-ext sysctl -w net.ipv6.conf.all.forwarding=1
docker exec -it ubuntu-ext sysctl -w net.ipv4.conf.all.forwarding=1
docker exec -it ubuntu-ext sysctl -w net.ipv6.conf.all.accept_dad=0

vlan_id="1500"
parent_if_name="eth0"
vlan_if_name="vxlan-ext"
bridge_if_name="bridge0-ext"

ip="192.168.1.150/16"
ip6="1500:1::150/16"

docker exec -it ubuntu-ext ip link add $bridge_if_name type bridge
docker exec -it ubuntu-ext ip link set $bridge_if_name up
docker exec -it ubuntu-ext ip addr add $ip dev $bridge_if_name
docker exec -it ubuntu-ext ip addr add $ip6 dev $bridge_if_name

for index in {0..5}
do
    id=$((index + 1))
    vni=$((vlan_id + id))
    if_name="${vlan_if_name}-${id}"

    docker exec -it ubuntu-ext ip link add $if_name type vxlan id $vni group 239.1.$id.1 dev $parent_if_name dstport 4789
    docker exec -it ubuntu-ext ip link set $if_name master $bridge_if_name
    docker exec -it ubuntu-ext ip link set $if_name up

    echo "if name: $if_name ; vni: $vni ; ip: $ip ; ipv6: $ip6"
done


docker exec -it ubuntu-ext ip route replace 20.0.0.1/32 nexthop via 192.168.1.1 nexthop via 192.168.2.1
docker exec -it ubuntu-ext ip route replace 2000::1/128 nexthop via 1500:1::1 nexthop via 1500:2::1
