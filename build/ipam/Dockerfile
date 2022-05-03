ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest

FROM golang:1.18.1 as build

ENV GO111MODULE=on
ARG meridio_version=0.0.0-unknown

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o ipam ./cmd/ipam

FROM ${base_image}
WORKDIR /root/
COPY --from=build /app/ipam .
CMD ["./ipam"]
