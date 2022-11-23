#! /bin/sh
##
## meridio-init.sh --
##
##   This script should run in an initContainer and prepare for
##   various Meridio PODs.
##
##   Tunnel handling copied from;
##   https://github.com/Nordix/k8s-service-tunnel
##
##   Config:
##
##   Parameter:   Environment var:  Description:
##   --dev=       TUNNEL_DEV        Tunnel device name (vxlan0)
##   --master=    TUNNEL_MASTER     Tunnel master device (eth0)
##   --peer=      TUNNEL_PEER       Ip address of the remote side
##   --id=        TUNNEL_ID         VNI for vxlan (333)
##   --dport=     TUNNEL_DPORT      Port for the remote side (5533)
##   --sport=     TUNNEL_SPORT      Port for the local side (5533)
##   --ipv4=      TUNNEL_IPV4       IPv4 on the tunnel device, e.g. 10.30.30.1/24
##   --ipv6=      TUNNEL_IPV6       IPv6 on the tunnel device, e.g. fd00:1::1/64
##   --ipv4-only  __ipv4_only       Only set sysctls for ipv4
##   --ipv6-only  __ipv6_only       Only set sysctls for ipv6
##
## Commands;
##

prg=$(basename $0)
dir=$(dirname $0); dir=$(readlink -f $dir)
tmp=/tmp/${prg}_$$

die() {
    echo "ERROR: $*" >&2
    rm -rf $tmp
    exit 1
}
cmd_help() {
    grep '^##' $0 | cut -c3-
    rm -rf $tmp
    exit 0
}

log() {
	echo "$prg: $*" >&2
}

# initvar <variable> [default]
#   Initiate a variable. The __<variable> will be defined if not set,
#   from $TUNNEL_<variable-upper-case> or from the passed default
initvar() {
	local n N v
	n=$1
	v=$(eval "echo \$__$n")
	test -n "$v" && return 0	# Already set
	N=$(echo $n | tr a-z A-Z)
	v=$(eval "echo \$TUNNEL_$N")
	if test -n "$v"; then
		eval "__$n=$v"
		return 0
	fi
	test -n "$2" && eval "__$n=$2"
	return 0
}

##   env
##     Print environment.
cmd_env() {
	test "$envset" = "yes" && return 0
	params="type|dev|master|peer|id|dport|sport|ipv4|ipv6|ipv6_only|ipv4_only"
	initvar dev vxlan0
	initvar master eth0
	initvar peer
	initvar id 333
	initvar dport 5533
	initvar sport 5533
	initvar ipv4
	initvar ipv6
	if test "$cmd" = "env"; then
		set | grep -E "^__($params).*=" | sort
		return 0
	fi
	envset=yes
}
##   hold
##     Don't return
cmd_hold() {
	log "Block return"
	tail -f /dev/null
}
##   wait_for_udp --peer=ip-address [--sport=]
##     Wait for an UDP packet. This MUST be done *before* setting up
##     the tunnel if the peer is connected through a K8s service. It
##     indicates that an UDP "connection" has been setup. If the
##     tunnel setup without a UDP connection messages will be sent to
##     the peer with the node IP as source (the normal ipv4 egress
##     setup in K8s). This will fail and worse, it will mess up the
##     conntracker so later connect attempts via the service will
##     fail!
cmd_wait_for_udp() {
	cmd_env
	log "Waiting for UDP packet from $__peer, port $__sport"
	test -n "$__peer" || die "No peer address"
	tcpdump -ni eth0 --immediate-mode -c 1 udp and host $__peer and port $__sport
}
##   vxlan --peer=ip-address [--master] [--dev=] [--id=vni] \
##       [--dport=] [--sport=]
##     Setup a vxlan tunnel. In a POD this should be preceded by a
##     "wait_for_udp".
cmd_vxlan() {
	cmd_env
	log "Setup a VXLAN tunnel to [$__peer]"
	test -n "$__peer" || die "No peer address"
	local sport1=$((__sport + 1))
	if test -n "$__master"; then
		ip link add $__dev type vxlan id $__id dev $__master remote $__peer \
			dstport $__dport srcport $__sport $sport1
	else
		ip link add $__dev type vxlan id $__id remote $__peer \
			dstport $__dport srcport $__sport $sport1
	fi
}
##   lb
##     Setup for Meridio stateless-lb. If $TUNNEL_PEER is specified
##     a tunnel is setup.
cmd_lb() {
	log "Setup for Meridio stateless-lb"
	if test "$__ipv6_only" != "yes"; then
		sysctl -w net.ipv4.conf.all.forwarding=1
		sysctl -w net.ipv4.fib_multipath_hash_policy=1
		sysctl -w net.ipv4.conf.all.rp_filter=0
		sysctl -w net.ipv4.conf.default.rp_filter=0
	fi
	if test "$__ipv4_only" != "yes"; then
		sysctl -w net.ipv6.fib_multipath_hash_policy=1
		sysctl -w net.ipv6.conf.all.forwarding=1
	fi
	cmd_env
	test -n "$__peer" || return 0

	cmd_wait_for_udp || log "FAILED: wait_for_udp"
	cmd_vxlan
	ip link set up dev $__dev
	test -n "$__ipv4" && ip addr add $__ipv4 dev $__dev
	test -n "$__ipv6" && ip -6 addr add $__ipv6 dev $__dev	
	return 0
}
##   proxy
##     Setup for Meridio proxy
cmd_proxy() {
	log "Setup for Meridio proxy"
	if test "$__ipv6_only" != "yes"; then
		sysctl -w net.ipv4.conf.all.forwarding=1
		sysctl -w net.ipv4.fib_multipath_hash_policy=1
		sysctl -w net.ipv4.conf.all.rp_filter=0
		sysctl -w net.ipv4.conf.default.rp_filter=0
	fi
	if test "$__ipv4_only" != "yes"; then
		sysctl -w net.ipv6.conf.all.forwarding=1
		sysctl -w net.ipv6.conf.all.accept_dad=0
		sysctl -w net.ipv6.fib_multipath_hash_policy=1
	fi
}
##   tapa
##     Setup for Meridio TAPA
cmd_tapa() {
	log "Setup for Meridio tapa"
	sysctl -w net.ipv4.conf.all.arp_announce=2
}


##
if test -n "$1"; then
	cmd=$1
	shift
else
	cmd=$INIT_FUNCTION
fi
test -n "$cmd" || die 'No command. Try "help"'

# Get the command
grep -q "^cmd_$cmd()" $0 $hook || die "Invalid command [$cmd]"

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

# Execute command
trap "die Interrupted" INT TERM
cmd_$cmd "$@"
status=$?
rm -rf $tmp

if test $status -eq 0; then
   test "$INIT_EXIT" = "NO" && cmd_hold
fi
exit $status
