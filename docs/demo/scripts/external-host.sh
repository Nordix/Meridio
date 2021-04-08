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

vlan_id="1500"
parent_if_name="eth0"
vlan_if_name="vxlan-ext"

for index in {0..5}
do
    id=$((index + 1))
    vni=$((vlan_id + id))
    if_name="${vlan_if_name}-${id}"
    ip="192.168.${id}.150/24"

    docker exec -it ubuntu-ext ip link add $if_name type vxlan id $vni group 239.1.$id.1 dev $parent_if_name dstport 4789
    docker exec -it ubuntu-ext ip addr add $ip dev $if_name
    docker exec -it ubuntu-ext ip link set $if_name up

    echo "if name: $if_name ; vni: $vni ; ip: $ip"
done
