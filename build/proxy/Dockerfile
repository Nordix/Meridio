ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest
ARG USER=meridio
ARG UID=10005
ARG HOME=/home/${USER}

FROM golang:1.23 as build

ARG meridio_version=0.0.0-unknown
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o proxy ./cmd/proxy

FROM ${base_image}
ARG USER
ARG UID
ARG HOME
RUN addgroup --gid $UID $USER \
  && adduser $USER --home $HOME --uid $UID -G $USER --disabled-password \
  && chown -R :root "${HOME}" && chmod -R g+s=u "${HOME}"
WORKDIR $HOME
COPY --from=build /app/proxy .
RUN setcap 'cap_net_admin+ep' ./proxy
USER ${UID}:${UID}
CMD ["./proxy"]
