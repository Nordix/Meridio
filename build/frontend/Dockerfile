FROM golang:1.16 as build

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOBIN=/bin
ARG meridio_version=0.0.0-unknown

WORKDIR /app
COPY go.mod .
RUN go mod tidy
RUN go mod download
RUN go get github.com/grpc-ecosystem/grpc-health-probe@v0.4.2
COPY . .
RUN go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o frontend ./cmd/frontend


FROM alpine:edge

RUN apk update && apk add bash net-tools iproute2 procps vim less tcpdump iputils bird
RUN mkdir -p /run/bird && \
	mkdir -p /etc/bird

WORKDIR /root
COPY --from=build /app/frontend ./
COPY --from=build /bin/grpc-health-probe /bin/grpc_health_probe

CMD ["./frontend"]
