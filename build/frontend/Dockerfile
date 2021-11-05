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


FROM alpine:edge

RUN apk update && apk add bash net-tools iproute2 procps vim less tcpdump iputils bird
RUN mkdir -p /run/bird && \
	mkdir -p /etc/bird

WORKDIR /root
COPY --from=build /app/frontend ./

CMD ["./frontend"]
