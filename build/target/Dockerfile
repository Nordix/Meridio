FROM golang:alpine as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o target ./cmd/target

FROM ubuntu:20.10

RUN apt-get update -y
RUN apt-get install -y iproute2 tcpdump iptables net-tools iputils-ping ipvsadm netcat

WORKDIR /root/
COPY --from=build /app/target .

CMD ./target
