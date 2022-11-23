#! /bin/sh
##
## build.sh --
##   Build script for https://github.com/Nordix/Meridio.
##
##   This script builds the Meridio images using the go environment on
##   your host, as opposed to build in Docker as the Makefile
##   does. This simplify repeated builds for development and tests.
##   And it's way faster!
##
## Commands;
##

prg=$(basename $0)
dir=$(dirname $0); dir=$(readlink -f $dir); dir=$(readlink -f $dir/..)
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
	test -n "$__out" || __out=$(readlink -f $dir/_output)
	test -n "$__targets" || __targets="stateless-lb proxy tapa ipam nsp frontend"
	test -n "$__registry" || __registry=registry.nordix.org/cloud-native/meridio
	test -n "$__version" || __version=local
	test -n "$__nfqlb" || __nfqlb=1.0.0
	test -n "$ARCHIVE" || ARCHIVE=$HOME/Downloads
	test "$cmd" = "env" && set | grep -E '^(__.*|ARCHIVE)='
}
##  init_image
##    Build an image used for initContainers
cmd_init_image() {
	cmd_env
	local tag=$__registry/init:$__version
	local dockerfile=
	mkdir -p $tmp/root
	cp $dir/hack/meridio-init.sh $tmp/root
	sed -e "s,/start-command,/meridio-init.sh," \
		< $dir/hack/host-build/Dockerfile.default > $tmp/Dockerfile
	docker build -t $tag $tmp || die "docker build $base"
	echo $tag
}
##  base_image
##    Build the base image
cmd_base_image() {
	cmd_env
	local base=$(grep base_image= $dir/hack/host-build/Dockerfile.default | cut -d= -f2)
	local dockerfile=$dir/build/base-image/Dockerfile
	mkdir -p $tmp
	docker build -t $base -f $dockerfile $tmp || die "docker build $base"
	echo $base
}
##  binaries [targets...]
##    Build binaries. Build in Meridio/_output
cmd_binaries() {
	cmd_env
	cd $dir
	mkdir -p $__out
	test -n "$1" && __targets="$@"
	local gitver=$(git describe --dirty --tags)
	local n cmds cgo
	for n in $__targets; do
		if echo $n | grep -qE 'ipam|nsp'; then
			# Requires CGO_ENABLED=1
			cgo="$cgo $dir/cmd/$n"
		else
			cmds="$cmds $dir/cmd/$n"
		fi
	done
	if test -n "$cmds"; then
		CGO_ENABLED=0 GOOS=linux go build -o $__out \
			-ldflags "-extldflags -static -X main.version=$gitver" $cmds \
			|| die "go build $cmds"
	fi
	if test -n "$cgo"; then
		mkdir -p $tmp
		if ! CGO_ENABLED=1 GOOS=linux go build -o $__out \
			-ldflags "-extldflags -static -X main.version=$gitver" \
			$cgo > $tmp/out 2>&1; then
			cat $tmp/out
			die "go build $cgo"
		fi
	fi
	strip $__out/*
}
##  images [--registry=] [--version=] [targets...]
##    Build docker images.
cmd_images() {
	test -n "$1" && __targets="$@"
	cmd_binaries
	local n dockerfile x
	for n in $__targets; do
		x=$__out/$n
		test -x $x || die "Not built [$x]"
		rm -rf $tmp; mkdir -p $tmp/root
		cp $x $tmp/root
		if test "$n" = "load-balancer"; then
			mkdir -p $ARCHIVE
			local ar=$ARCHIVE/nfqlb-$__nfqlb.tar.xz
			if ! test -r $ar; then
				local url=https://github.com/Nordix/nfqueue-loadbalancer/releases/download
				curl -L $url/$__nfqlb/nfqlb-$__nfqlb.tar.xz > $ar || die Curl
			fi
			tar -C $tmp --strip-components=1 -xf $ar nfqlb-$__nfqlb/bin/nfqlb \
				|| die "tar $ar"
		fi
		dockerfile=$dir/hack/host-build/Dockerfile.$n
		test -r $dockerfile \
			|| dockerfile=$dir/hack/host-build/Dockerfile.default
		sed -e "s,/start-command,/$n," < $dockerfile > $tmp/Dockerfile
		docker build -t $__registry/$n:$__version $tmp \
			|| die "docker build $n"
	done
}

##
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
