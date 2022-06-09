FROM ubuntu:21.04

RUN apt-get update -y --fix-missing\
  && apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat wget bird2 ethtool conntrack xz-utils

WORKDIR /root/

ADD https://github.com/Nordix/ctraffic/releases/download/v1.7.0/ctraffic.gz ctraffic.gz
RUN gunzip ctraffic.gz \
  && chmod u+x ctraffic \
  && mv ctraffic /usr/bin/

ADD https://github.com/Nordix/mconnect/releases/download/v2.2.0/mconnect.xz mconnect.xz
RUN unxz mconnect.xz \
  && chmod u+x mconnect \
  && mv mconnect /usr/bin/ \
  && mkdir -p /etc/bird/ \
  && mkdir -p /run/bird

COPY docs/demo/scripts/kind/bird/bird-common.conf /etc/bird/
COPY docs/demo/scripts/kind/bird/bird-gw.conf /etc/bird/

CMD sleep 5 ; /usr/sbin/bird -d -c /etc/bird/bird-gw.conf