#! /bin/sh
##
## forwarder-test.sh --
##
##   Help script for the xcluster ovl/forwarder-test.
##
## Commands;
##

prg=$(basename $0)
dir=$(dirname $0); dir=$(readlink -f $dir)
me=$dir/$prg
tmp=/tmp/${prg}_$$

die() {
    echo "ERROR: $*" >&2
    rm -rf $tmp
    exit 1
}
help() {
    grep '^##' $0 | cut -c3-
    rm -rf $tmp
    exit 0
}
test -n "$1" || help
echo "$1" | grep -qi "^help\|-h" && help

log() {
	echo "$prg: $*" >&2
}
dbg() {
	test -n "$__verbose" && echo "$prg: $*" >&2
}

##  env
##    Print environment.
##
cmd_env() {
	test "$env_set" = "yes" && return 0

	if test "$cmd" = "env"; then
		set | grep -E '^(__.*)='
		return 0
	fi

	test -n "$xcluster_NSM_FORWARDER" || export xcluster_NSM_FORWARDER=vpp
	test -n "$xcluster_FIRST_WORKER" || export xcluster_FIRST_WORKER=1
	if test "$xcluster_FIRST_WORKER" = "1"; then
		export __mem1=4096
		test -n "$__nvm" || __nvm=3
		test "$__nvm" -gt 3 && __nvm=3
		export __mem=3072
	else
		export __mem1=1024
		test -n "$__nvm" || __nvm=4
		test "$__nvm" -gt 4 && __nvm=4
		export __mem=4096
	fi
	export __nvm
	test -n "$xcluster_DOMAIN" || xcluster_DOMAIN=xcluster
	test -n "$XCLUSTER" || die 'Not set [$XCLUSTER]'
	test -x "$XCLUSTER" || die "Not executable [$XCLUSTER]"
	eval $($XCLUSTER env)
	env_set=yes
}

##   test --list
##   test [--xterm] [--no-stop] [--local] [--nsm-local] [test...] > logfile
##     Exec tests
##
cmd_test() {
	if test "$__list" = "yes"; then
		grep '^test_' $me | cut -d'(' -f1 | sed -e 's,test_,,'
		return 0
	fi

	cmd_env
	start=starts
	test "$__xterm" = "yes" && start=start
	rm -f $XCLUSTER_TMP/cdrom.iso

	if test -n "$1"; then
		for t in $@; do
			test_$t
		done
	else
		test_trench
	fi

	now=$(date +%s)
	tlog "Xcluster test ended. Total time $((now-begin)) sec"

}

test_start_empty() {
	__mode=dual-stack
	export xcluster___mode=$__mode
	xcluster_prep $__mode
	export TOPOLOGY=multilan
	. $($XCLUSTER ovld network-topology)/$TOPOLOGY/Envsettings
	export __smp202=3
	export __nets202=0,1,2,3,4,5
	echo "--nvm=$__nvm --mem1=$__mem1 --mem=$__mem"
	# Avoid "Illegal instruction" error (vpp)
	export __kvm_opt='-M q35,accel=kvm,kernel_irqchip=split -object rng-random,filename=/dev/urandom,id=rng0 -device virtio-rng-pci,rng=rng0,max-bytes=1024,period=80000 -cpu host'
	# Required by the vpp-forwarder but not used without dpdk
	export __append1="hugepages=128"
	export __append2="hugepages=128"
	export __append3="hugepages=128"
	xcluster_start network-topology spire k8s-pv nsm-ovs $@ forwarder-test

	otc 1 check_namespaces
	otc 1 check_nodes
}

##   test start
##     Start the cluster with NSM. Default; xcluster_NSM_FORWARDER=vpp
test_start() {
	tcase "Start with NSM, forwarder=$xcluster_NSM_FORWARDER"
	test_start_empty $@
	otc 202 "conntrack 20000"
	otcw "conntrack 20000"
	test "$__use_multus" = "yes" && otc 1 multus_setup
	otcprog=spire_test
	otc 1 start_spire_registrar
	otcprog=nsm-ovs_test
	local vm
	for vm in $(seq $xcluster_FIRST_WORKER $__nvm); do
		otc $vm "ifup eth2"
		otc $vm "ifup eth3"
	done
	otc 1 start_nsm
	otc 1 start_forwarder
	test "$xcluster_NSM_FORWARDER" = "vpp" && otc 1 vpp_version
	unset otcprog
}

##   test [--trenches=red,...] [--use-multus] trench (default)
##     Test trenches. The default is to test all 3 trenches
test_trench() {
	local x
	test "$__use_multus" = "yes" && export __use_multus
	if test "$__local" = "yes"; then
		x="images:local"
		export __local
	fi
	test -n "$__trenches" || __trenches=red,blue,green
	test "$__nsm_local" = "yes" && export nsm_local=yes
	tlog "=== forwarder-test: Test trenches [$__trenches] $x"
	test_start
	local trench
	for trench in $(echo $__trenches | tr , ' '); do
		trench_test $trench
	done
	tcase "Re-test connectivity with all trenches"
	for trench in $(echo $__trenches | tr , ' '); do
		otc 202 "mconnect $trench"
	done
	xcluster_stop
}

cmd_add_trench() {
	test -n "$1" || die 'No trench'
	case $1 in
		red) otc 202 "setup_vlan --tag=100 eth3";;
		blue) otc 202 "setup_vlan --tag=200 eth3";;
		green) otc 202 "setup_vlan --tag=100 eth4";;
		*) tdie "Invalid trench [$1]";;
	esac
	otc 1 "trench --local=$__local $1"
}

cmd_add_multus_trench() {
	cmd_env
	case $1 in
		red)
			otcw "local_vlan --tag=100 eth2"
			otc 202 "setup_vlan --tag=100 eth3";;
		blue)
			otcw "local_vlan --tag=200 eth2"
			otc 202 "setup_vlan --tag=200 eth3";;
		green)
			otcw "local_vlan --tag=100 eth3"
			otc 202 "setup_vlan --tag=100 eth4";;
		*) tdie "Invalid trench [$1]";;
	esac
	otc 1 "trench_multus --local=$__local $1"
}

trench_test() {
	if test "$__use_multus" = "yes"; then
		cmd_add_multus_trench $1
	else
		cmd_add_trench $1
	fi
	otc 202 "collect_lb_addresses $1"
	otc 202 "trench_vip_route $1"
	tcase "Sleep 10 sec..."
	sleep 10
	otc 202 "mconnect $1"
}

##   test [--cnt=n] scale
##     Scaling targets. By changing replicas and by disconnect targets
##     from the stream.
test_scale() {
	if test "$__local" = "yes"; then
		x="images:local"
		export __local
	fi
	test -n "$__cnt" || __cnt=1
	tlog "=== forwarder-test: Scale target cnt=$__cnt $x"
	test_start
	local trench=red
	trench_test red
	otc 1 "scale $trench 8"
	otc 1 "check_targets $trench 8"
	while test $__cnt -gt 0; do
		tlog "cnt=$__cnt"
		__cnt=$((__cnt - 1))
		otc 1 "disconnect_targets $trench 3"
		otc 1 "check_targets $trench 5"
		otc 202 "check_connections $trench 5"
		otc 1 "reconnect_targets $trench"
		otc 1 "check_targets $trench 8"
		otc 202 "check_connections $trench 8"
	done
	otc 1 "scale $trench 4"
	otc 1 "check_targets $trench 4"
	xcluster_stop
}


##   test [--nsm-local] nsm
##     Test without meridio but with NSM in a "meridio alike" way,
##     i.e. NSE and NSC in separate K8s namespaces.
test_nsm() {
	tlog "=== forwarder-test: NSM without Meridio"
	test "$__nsm_local" = "yes" && export nsm_local=yes
	test_start
	otc 1 "nsm red"
	otc 1 "nsm blue"
	otc 1 "nsm green"
	otc 202 "setup_vlan --tag=100 eth3"
	otc 202 "setup_vlan --tag=200 eth3"
	otc 202 "setup_vlan --tag=100 eth4"
	local ns
	if test $xcluster_FIRST_WORKER -gt 1 ; then
		# Some CNI-plugins takes a lot of juice :-(
		tcase "Sleep 5 sec ..."
		sleep 5
	fi
	for ns in red blue green; do
		otc 202 "collect_nsc_addresses $ns"
		otc 202 "ping_nsc_addresses $ns"
	done
	xcluster_stop
}

##   test multus
##     Test Multus setup without NSM or Meridio
test_multus() {
	tlog "=== forwarder-test: Multus without NSM or Meridio"
	export __use_multus=yes
	test_start_empty
	otc 1 multus_setup
	otcw "local_vlan --tag=100 eth2"
	otcw "local_vlan --tag=200 eth2"
	otcw "local_vlan --tag=100 eth3"
	otc 202 "setup_vlan --tag=100 eth3"
	otc 202 "setup_vlan --tag=200 eth3"
	otc 202 "setup_vlan --tag=100 eth4"
	local ns
	for ns in red blue green; do
		otc 1 "multus $ns"
		otc 202 "collect_alpine_addresses $ns"
		otc 202 "ping_alpine_addresses $ns"
	done
	xcluster_stop
}

##
##   generate_manifests [--dst=/tmp/$USER/meridio-manifests]
##     Generate manifests from Meridio helm charts.
cmd_generate_manifests() {
	unset KUBECONFIG
	test -n "$__dst" || __dst=/tmp/$USER/meridio-manifests
	mkdir -p $__dst
	test -n "$__meridio_dir" || __meridio_dir=$(readlink -f ../../../../../..)
	local m
	m=$__meridio_dir/deployments/helm
	test -d $m || die "Not a directory [$m]"
	helm template --generate-name $m > $__dst/meridio.yaml
	m=$__meridio_dir/examples/target/helm
	test -d $m || die "Not a directory [$m]"
	helm template --generate-name $m > $__dst/target.yaml
	echo "Manifests generated in [$__dst]"
}

##   build_image [images...]
##     Build local images and upload to the local registry.
cmd_build_image() {
	export meridio_version=$(git describe --dirty --tags)
	echo "meridio_version=$meridio_version"
	local images=$($XCLUSTER ovld images)/images.sh
	test -x $images || dir "Can't find ovl/images/images.sh"
	local tagbase=registry.nordix.org/cloud-native/meridio
	local i d ver
	if test -n "$1"; then
		for i in $@; do
			test -d images/$i || die "Not a directory [images/$i]"
			ver=local
			echo $1 | grep -q meridio-app && ver=xcluster
			$images mkimage --upload --strip-host --tag=$tagbase/$i:$ver images/$i
		done
	else
		for d in $(find images -mindepth 1 -maxdepth 1 -type d); do
			i=$(basename $d)
			ver=local
			echo $i | grep -q meridio-app && ver=xcluster
			echo "=== Building [$i:$ver]"
			$images mkimage --upload --strip-host --tag=$tagbase/$i:$ver $d
		done
	fi
}



##
. $($XCLUSTER ovld test)/default/usr/lib/xctest
indent=''

# Get the command
cmd=$1
shift
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
exit $status
