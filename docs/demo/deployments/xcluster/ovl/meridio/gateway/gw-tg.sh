#! /bin/sh

die() {
    echo "ERROR: $*" >&2
    exit 1
}

#------------------------------------------------------------------------------------

# Parse options

# --tg4_addr=[IPv4 addr]
# --tg6_addr=[IPv6 addr]
# --bird_conf=[/etc/bird/FILE.conf]
# --rt=[routing table id]
# "unlabelled" options are passed to BIRD

while echo "$1" | grep -q '^--'; do
    if echo $1 | grep -q =; then
        o=$(echo "$1" | cut -d= -f1 | sed -e 's,-,_,g')
        v=$(echo "$1" | cut -d= -f2-)
        echo "opt: $o"=\"$v\"
        eval "$o=\"$v\""
        echo "$o=\"$v\""
    else
        o=$(echo "$1" | sed -e 's,-,_,g')
        echo "opt: $o"=\"yes\"
        eval "$o=yes"
        echo "$o=yes"
    fi
    shift
done
unset o v
long_opts=`set | grep '^__' | cut -d= -f1`

#------------------------------------------------------------------------------------

# BIRD config file to use
test -n "$__bird_conf" || __bird_conf="/etc/bird/bird-gw.conf"

# set GW addresses in BIRD config if specified
if test -n "$__tg4_addr"; then
    echo "tg4: $__tg4_addr"
	sed -i -e "s/^define TG_ADDR4 =.*/define TG_ADDR4 = $__tg4_addr;/" $__bird_conf
fi
if test -n "$__tg6_addr"; then
    echo "tg4: $__tg6_addr"
	sed -i -e "s/^define TG_ADDR6 =.*/define TG_ADDR6 = $__tg6_addr;/" $__bird_conf
fi

if test -n "$__ext_if"; then
	sed -i -e "s|\(\s*\)interface \"[0-9a-zA-Z-]\+|\1interface \"$__ext_if|g" $__bird_conf
fi


#cleanup
#init
trap "cleanup && die Interrupted" INT TERM
echo "/usr/sbin/bird -f $@ -c $__bird_conf"
/usr/sbin/bird -f $@ -c $__bird_conf

status=$?
#cleanup
exit $status
