FROM alpine:edge

RUN apk update && apk add bash net-tools iproute2 procps vim less tcpdump iputils bird

RUN mkdir -p /run/bird \
	mkdir -p /etc/bird \
	mkdir -p /tmp
COPY bird-common.conf bird-gw.conf /etc/bird/
COPY gw.sh /tmp/
COPY bird-tg.conf /etc/bird/

ENTRYPOINT ["/usr/sbin/bird", "-f"]
CMD []
