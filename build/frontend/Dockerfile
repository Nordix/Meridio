ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest
ARG USER=meridio
ARG UID=10001
ARG HOME=/home/${USER}

FROM golang:1.21 as build

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
ARG USER
ARG UID
ARG HOME
RUN apk add bird
RUN mkdir -p /run/bird && mkdir -p /etc/bird
RUN addgroup --gid $UID $USER \
  && adduser $USER --home $HOME --uid $UID -G $USER --disabled-password \
  && chown -R :root "${HOME}" && chmod -R g+s=u "${HOME}"
WORKDIR $HOME
COPY --from=build /app/frontend ./
# note: File permissions of unix spire-agent-socket grant "write" access for "others",
# thus cap_dac_override is not required by this hostPath.
# Permissions for logging to file (bird) and interaction between bird and frontend
# can be secured by writable volume mounts and by usage of "fsGroup".
RUN setcap 'cap_net_admin+ep' ./frontend \
  && setcap 'cap_net_admin,cap_net_bind_service,cap_net_raw+ep' /usr/sbin/bird
USER ${UID}:${UID}
CMD ["./frontend"]
