FROM ubuntu:22.04

RUN apt-get update -y --fix-missing \
  && apt-get install -y iproute2 tcpdump net-tools iputils-ping netcat wget screen xz-utils strace nftables ipvsadm iptables bird2 ethtool conntrack dnsutils jq

ADD https://github.com/Nordix/ctraffic/releases/download/v1.7.0/ctraffic.gz ctraffic.gz
RUN gunzip ctraffic.gz \
  && chmod a+x ctraffic

ADD https://github.com/Nordix/mconnect/releases/download/v2.2.0/mconnect.xz mconnect.xz
RUN unxz mconnect.xz \
  && chmod a+x mconnect

ADD https://github.com/Nordix/nfqueue-loadbalancer/releases/download/1.0.0/nfqlb-1.0.0.tar.xz /
RUN tar --strip-components=1 -xf /nfqlb-1.0.0.tar.xz nfqlb-1.0.0/bin/nfqlb

ADD https://github.com/kubernetes-sigs/cri-tools/releases/download/v1.24.1/crictl-v1.24.1-linux-amd64.tar.gz /
RUN tar zxvf crictl-v1.24.1-linux-amd64.tar.gz -C /bin
RUN rm -f crictl-v1.24.1-linux-amd64.tar.gz

CMD ["tail", "-f", "/dev/null"]