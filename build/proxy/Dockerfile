ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest

FROM golang:1.18.1 as build

ARG meridio_version=0.0.0-unknown
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o proxy ./cmd/proxy

FROM ${base_image}
WORKDIR /root/
COPY --from=build /app/proxy .
CMD ["./proxy"]
