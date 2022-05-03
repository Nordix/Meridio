FROM alpine:3.15
RUN apk update && apk add iproute2 tcpdump iputils
ADD https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.2/grpc_health_probe-linux-amd64 /bin/grpc_health_probe
RUN chmod a+x /bin/grpc_health_probe
