ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest

FROM golang:1.18.1 as build

ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOBIN=/bin
ARG meridio_version=0.0.0-unknown

WORKDIR /app
COPY go.mod .
RUN go mod tidy
RUN go mod download
COPY . .
RUN go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o frontend ./cmd/frontend


FROM ${base_image}
RUN apk add bird
RUN mkdir -p /run/bird && mkdir -p /etc/bird

WORKDIR /root
COPY --from=build /app/frontend ./
CMD ["./frontend"]
