FROM golang:alpine as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o load-balancer ./cmd/load-balancer

FROM ubuntu:20.10 as lb-builder

RUN apt-get update -y --fix-missing
RUN apt-get install -y build-essential
run apt-get install -y libmnl-dev
run apt-get install -y libnetfilter-queue-dev

COPY ./third_party/lb /lb-src
WORKDIR /lb-src
RUN gcc -o /lb lb.c maglev.c -lmnl -lnetfilter_queue -lrt

FROM ubuntu:20.10

RUN apt-get update -y
run apt-get install -y libmnl-dev
run apt-get install -y libnetfilter-queue-dev
RUN apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm  

WORKDIR /root/
COPY --from=build /app/load-balancer .
COPY --from=lb-builder /lb /bin/lb

CMD ./load-balancer
