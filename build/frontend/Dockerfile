FROM golang:1.16 as build

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app
COPY go.mod .
RUN go mod tidy
RUN go mod download
COPY . .
RUN go build -ldflags '-extldflags "-static"' -o frontend ./cmd/frontend


FROM debian:bullseye-slim

RUN apt-get update && DEBIAN_FRONTEND=noninteractive \
	apt-get install --no-install-recommends -y net-tools && \
	apt-get install --no-install-recommends -y iproute2 && \
	apt-get install --no-install-recommends -y procps && \
	apt-get install --no-install-recommends -y vim && \
	apt-get install --no-install-recommends -y less && \
	apt-get install --no-install-recommends -y tcpdump && \
	apt-get install --no-install-recommends -y iputils-ping && \
	apt-get install --no-install-recommends -y bird2

RUN mkdir -p /run/bird && \
	mkdir -p /etc/bird

WORKDIR /root
COPY --from=build /app/frontend ./

CMD ["./frontend"]
