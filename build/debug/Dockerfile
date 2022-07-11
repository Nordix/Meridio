FROM alpine:3.15
RUN apk update && apk add iproute2 tcpdump iputils net-tools ethtool nftables \
  && setcap 'cap_sys_ptrace,cap_dac_override+ep' /sbin/ss \
  && setcap 'cap_sys_ptrace,cap_dac_override+ep' /bin/netstat \
  && setcap 'cap_net_raw+ep' /bin/ping \
  && setcap 'cap_net_raw+ep' /usr/bin/tcpdump \
  && setcap 'cap_net_admin+ep' /sbin/ip \
  && setcap 'cap_net_admin+ep' /usr/sbin/nft \
  && chmod u+s /usr/sbin/ethtool