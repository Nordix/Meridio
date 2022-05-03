FROM golang:1.18.1 as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o target-client ./test/applications/target-client

FROM ubuntu:21.04

RUN apt-get update -y --fix-missing \
  && apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat wget screen conntrack xz-utils

WORKDIR /root/

ADD https://github.com/Nordix/ctraffic/releases/download/v1.7.0/ctraffic.gz ctraffic.gz
RUN gunzip ctraffic.gz \
  && chmod u+x ctraffic

ADD https://github.com/Nordix/mconnect/releases/download/v2.2.0/mconnect.xz mconnect.xz
RUN unxz mconnect.xz \
  && chmod u+x mconnect

COPY --from=build /app/target-client .

CMD ./ctraffic -server -address [::]:5000
