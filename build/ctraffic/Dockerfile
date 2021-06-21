FROM golang:1.16 as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o target-client ./test/applications/target-client

FROM ubuntu:20.10

RUN apt-get update -y --fix-missing
RUN apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat wget

WORKDIR /root/

RUN wget https://github.com/Nordix/ctraffic/releases/download/v1.3.0/ctraffic.gz
RUN gunzip ctraffic.gz 
RUN chmod u+x ctraffic

COPY --from=build /app/target-client .

CMD ./target-client connect -ns load-balancer ; ./target-client request -ns load-balancer ; ./ctraffic -server -address [::]:5000