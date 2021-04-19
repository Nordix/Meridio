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

docker exec -it ubuntu-ext sysctl -w net.ipv4.fib_multipath_hash_policy=1
docker exec -it ubuntu-ext sysctl -w net.ipv6.fib_multipath_hash_policy=1

parent_if_name="eth0"
vlan_if_name="vlan0"
vlan_id="101"

docker exec -it ubuntu-ext ip link add link $parent_if_name name $vlan_if_name type vlan id $vlan_id
docker exec -it ubuntu-ext ip link set $vlan_if_name up

docker exec -it ubuntu-ext ip addr add 169.254.0.0/32 dev $vlan_if_name
docker exec -it ubuntu-ext ip addr add 169.254.0.2/32 dev $vlan_if_name
docker exec -it ubuntu-ext ip addr add 169.254.0.4/32 dev $vlan_if_name
docker exec -it ubuntu-ext ip addr add 169.254.0.6/32 dev $vlan_if_name

docker exec -it ubuntu-ext ip route add 169.254.0.1 dev vlan0 src 169.254.0.0
docker exec -it ubuntu-ext ip route add 169.254.0.3 dev vlan0 src 169.254.0.2
docker exec -it ubuntu-ext ip route add 169.254.0.5 dev vlan0 src 169.254.0.4
docker exec -it ubuntu-ext ip route add 169.254.0.7 dev vlan0 src 169.254.0.6

docker exec -it ubuntu-ext ip route replace 20.0.0.1/32 nexthop via 169.254.0.1 nexthop via 169.254.0.3
