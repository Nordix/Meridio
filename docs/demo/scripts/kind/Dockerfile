FROM ubuntu:20.10

RUN apt-get update -y --fix-missing
RUN apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat wget bird2

WORKDIR /root/

RUN wget https://github.com/Nordix/ctraffic/releases/download/v1.3.0/ctraffic.gz
RUN gunzip ctraffic.gz 
RUN chmod u+x ctraffic
RUN mv ctraffic /usr/bin/

RUN mkdir -p /etc/bird/
RUN mkdir -p /run/bird

COPY docs/demo/scripts/kind/bird/bird-common.conf /etc/bird/
COPY docs/demo/scripts/kind/bird/bird-gw.conf /etc/bird/

CMD sleep 5 ; /usr/sbin/bird -d -c /etc/bird/bird-gw.conf