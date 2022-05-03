ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest

FROM golang:1.18.1 as build
ARG meridio_version=0.0.0-unknown
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o load-balancer ./cmd/load-balancer

FROM ubuntu:21.04 as lb-builder
WORKDIR /
RUN apt update -y --fix-missing
RUN apt install -y curl xz-utils
ADD https://github.com/Nordix/nfqueue-loadbalancer/releases/download/1.0.0/nfqlb-1.0.0.tar.xz /
RUN tar --strip-components=1 -xf /nfqlb-1.0.0.tar.xz nfqlb-1.0.0/bin/nfqlb

FROM ${base_image}
WORKDIR /root/
COPY --from=build /app/load-balancer .
COPY --from=lb-builder /bin/nfqlb /bin/nfqlb
CMD ["./load-balancer"]
