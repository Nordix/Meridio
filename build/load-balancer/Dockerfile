ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest
ARG USER=meridio
ARG UID=10002
ARG HOME=/home/${USER}

FROM golang:1.19.0 as build
ARG meridio_version=0.0.0-unknown
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o load-balancer ./cmd/load-balancer

FROM ${base_image} as lb-builder
WORKDIR /
ADD https://github.com/Nordix/nfqueue-loadbalancer/releases/download/1.0.0/nfqlb-1.0.0.tar.xz /
RUN tar --strip-components=1 -xf /nfqlb-1.0.0.tar.xz nfqlb-1.0.0/bin/nfqlb

FROM ${base_image}
ARG USER
ARG UID
ARG HOME
RUN apk add nftables
RUN addgroup --gid $UID $USER \
  && adduser $USER --home $HOME --uid $UID -G $USER --disabled-password \
  && chown -R :root "${HOME}" && chmod -R g+s=u "${HOME}"
WORKDIR $HOME
COPY --from=build /app/load-balancer .
COPY --from=lb-builder /bin/nfqlb /bin/nfqlb
# cap_dac_override required by non-root user because of nsm-socket hostPath file permissions
# (while file permissions of hostPath unix spire-agent-socket grant "write" access for "others")
RUN setcap 'cap_net_admin,cap_dac_override+ep' ./load-balancer \
  && chown root:root /bin/nfqlb && setcap 'cap_net_admin,cap_ipc_lock,cap_ipc_owner+ep' /bin/nfqlb \
  && setcap 'cap_net_admin+ep' /usr/sbin/nft
USER ${UID}:${UID}
CMD ["./load-balancer"]
