FROM debian:bullseye-slim

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get update && \
	apt-get install -y net-tools && \
	apt-get install -y iproute2 && \
	apt-get install -y procps && \
	apt-get install -y vim && \
	apt-get install -y tcpdump && \
	apt-get install iputils-ping && \
	apt-get install -y bird2

RUN mkdir -p /run/bird \
	mkdir -p /etc/bird \
	mkdir -p /tmp
COPY bird-common.conf bird-gw.conf /etc/bird/
COPY gw.sh /tmp/
COPY bird-tg.conf /etc/bird/

ENTRYPOINT ["/usr/sbin/bird", "-f"]
CMD []
