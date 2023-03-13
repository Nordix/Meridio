FROM alpine:3.17.2
RUN apk update && apk add iproute2 tcpdump iputils net-tools libcap \
  && setcap 'cap_sys_ptrace,cap_dac_override+ep' /sbin/ss \
  && setcap 'cap_sys_ptrace,cap_dac_override+ep' /bin/netstat \
  && setcap 'cap_net_raw+ep' /bin/ping \
  && setcap 'cap_net_raw+ep' /usr/bin/tcpdump
ADD https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.15/grpc_health_probe-linux-amd64 /bin/grpc_health_probe
RUN chmod a+x /bin/grpc_health_probe
