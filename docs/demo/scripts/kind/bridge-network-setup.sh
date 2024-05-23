#!/bin/bash

die() {
    echo "ERROR: $*" >&2
    exit 1
}

# Parse parameters
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


ip_families=("ipv4" "ipv6" "dualstack")

test -n "$__network_name" || export __network_name=meridio-net
test -n "$__kubernetes_workers" || export __kubernetes_workers=2
test -n "$__bridge_name" || export __bridge_name="br-meridio"
test -n "$__mtu" || export __mtu=1500
# verify IP family is supported
if test -n "$__ipfamily"; then
    if [[ ! " ${ip_families[*]} " =~ " $__ipfamily " ]]; then
        die "IP family must be one of the following values: ${ip_families[*]}"
    fi
else
    export __ipfamily="dualstack"
fi
# It is assumed, that new network interface would appear as eth1
# on the workers once they get connected to the new docker network.
test -n "$__interface" || export __interface=eth1

if [ $__network_name == "kind" ]; then
    die "invalid breakout network: '$__network_name'"
fi


max_attempts=20
sleep_interval=3
attempt=0
# Create the new docker network and connect the kind workers.
# Network gets only created if does not exists. Thus, it must
# be removed beforehand if config options might change.
# Note: Probably not much point in enabling IPv6 or adding subnet as custom IPs will be
# configured separately anyways. Also, docker network seems to have an IPv4 subnet by default.
echo "Create docker network '$__network_name' of type bridge"
ipv6_option=""
if [[ ! " ipv4 " =~ " $__ipfamily " ]]; then
    # docker IPv6 address pool only yields a single subnet, so add it manually
    # (possible workaround would be to adjust default-address-pools config for dockerd to include multiple subnets)
    ipv6_option="--ipv6 --subnet fc00:dead:beef::/48 --gateway fc00:dead:beef::1"
fi
# assumed to appear as eth1 on workers
docker network create -d bridge $__network_name --opt com.docker.network.driver.mtu=$__mtu $ipv6_option || die "docker network create"

while [ $attempt -lt $max_attempts ]; do
    if docker network ls | grep -q "$__network_name"; then
        echo "Docker network '$__network_name' found"
        break
    else
        echo "Docker network '$__network_name' not found. Attempt $((attempt + 1)) of $max_attempts"
        attempt=$((attempt + 1))
        sleep $sleep_interval
    fi
done
if [ $attempt -ge $max_attempts ]; then
    die "docker network '$__network_name' not found after $max_attempts attempts"
fi

for number in $(seq 1 $__kubernetes_workers) ; do \
    if [ $number -eq 1 ]; then
        number=""
    fi
    echo "Connect 'kind-worker$number' to docker network '$__network_name'"
    docker network connect $__network_name kind-worker$number || die "connect worker $number to docker network $__network_name"
done
sleep 1

# Set up a VLAN aware linux bridge on all kind workers.
echo "Set up bridge '$__bridge_name'."
for number in $(seq 1 $__kubernetes_workers) ; do \
    if [ $number -eq 1 ]; then
      number=""
    fi
    docker exec kind-worker$number ip link del $__bridge_name > /dev/null 2>&1
    docker exec kind-worker$number ip link add name $__bridge_name type bridge || die "create bridge '$__bridge_name'"
    docker exec kind-worker$number ip link set dev $__bridge_name type bridge vlan_filtering 1 || die "enable bridge vlan filtering"
    docker exec kind-worker$number ip link set dev $__bridge_name up  || die "set link status up for dev '$__bridge_name'"
    docker exec kind-worker$number ip link set dev $__interface master $__bridge_name || die "add '$__interface' to bridge '$__bridge_name'"
    # Any VLAN tagged traffic might appear on the interface
    docker exec kind-worker$number bridge vlan add dev $__interface vid 2-4094  || die "enable all available VLANs on trunk interface '$__interface'"
done
