#! /bin/bash

ssh root@localhost -p 12502 -- wget https://github.com/Nordix/ctraffic/releases/download/v1.3.0/ctraffic.gz
ssh root@localhost -p 12502 -- gunzip ctraffic.gz 
ssh root@localhost -p 12502 -- chmod u+x ctraffic
ssh root@localhost -p 12502 -- mv ctraffic /usr/bin/

ssh root@localhost -p 12502 -- sysctl -w net.ipv6.conf.all.disable_ipv6=0
ssh root@localhost -p 12502 -- sysctl -w net.ipv4.fib_multipath_hash_policy=1
ssh root@localhost -p 12502 -- sysctl -w net.ipv6.fib_multipath_hash_policy=1
ssh root@localhost -p 12502 -- sysctl -w net.ipv6.conf.all.forwarding=1
ssh root@localhost -p 12502 -- sysctl -w net.ipv4.conf.all.forwarding=1
ssh root@localhost -p 12502 -- sysctl -w net.ipv6.conf.all.accept_dad=0

parent_if_name="eth1"
vlan_if_name="vlan"
vlan_id="100"

for index in {0..5}
do
    if_name="${vlan_if_name}-${index}"
    vi=$((vlan_id + index))
    ip="169.254.${vi}.150/24"
    ip6="100:${vi}::150/64"

    ssh root@localhost -p 12502 -- ip link add link $parent_if_name name $if_name type vlan id $vi
    ssh root@localhost -p 12502 -- ip link set $if_name up

    ssh root@localhost -p 12502 -- ip addr add $ip dev $if_name
    ssh root@localhost -p 12502 -- ip addr add $ip6 dev $if_name
done

ssh root@localhost -p 12502 -- ip route replace 20.0.0.1/32 nexthop via 169.254.100.1 nexthop via 169.254.100.2
ssh root@localhost -p 12502 -- ip route replace 2000::1/128 nexthop via 100:100::1 nexthop via 100:100::2
