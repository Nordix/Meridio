FROM ubuntu:20.10

RUN apt-get update -y --fix-missing
RUN apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat wget

RUN wget https://github.com/Nordix/ctraffic/releases/download/v1.3.0/ctraffic.gz
RUN gunzip ctraffic.gz 
RUN chmod u+x ctraffic

CMD ./ctraffic -server -address [::]:5000