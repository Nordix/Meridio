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

parent_if_name="eth0"
vlan_if_name="vlan"
vlan_id="100"

for index in {0..5}
do
    if_name="${vlan_if_name}-${index}"
    vi=$((vlan_id + index))
    ip="169.254.${vi}.150/24"
    ip6="100:${vi}::150/64"

    docker exec -it ubuntu-ext ip link add link $parent_if_name name $if_name type vlan id $vi
    docker exec -it ubuntu-ext ip link set $if_name up

    docker exec -it ubuntu-ext ip addr add $ip dev $if_name
    docker exec -it ubuntu-ext ip addr add $ip6 dev $if_name
done

docker exec -it ubuntu-ext ip route replace 20.0.0.1/32 nexthop via 169.254.100.2 nexthop via 169.254.100.3
docker exec -it ubuntu-ext ip route replace 2000::1/128 nexthop via 100:100::2 nexthop via 100:100::3
