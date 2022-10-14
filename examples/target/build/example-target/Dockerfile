ARG USER=tapa-user
ARG UID=10006
ARG HOME=/home/${USER}

FROM golang:1.19.0 as build

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

WORKDIR /app/examples/target

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o target-client ./cmd/target-client

FROM ubuntu:22.04

ARG USER
ARG UID
ARG HOME

RUN apt-get update -y --fix-missing \
  && apt-get install -y iproute2 tcpdump net-tools iputils-ping netcat wget screen xz-utils strace \
  && setcap 'cap_sys_ptrace,cap_dac_override+ep' /usr/bin/ss \
  && setcap 'cap_sys_ptrace,cap_dac_override+ep' /usr/bin/netstat \
  && setcap 'cap_net_raw+ep' /usr/bin/ping \
  && setcap 'cap_net_raw+ep' /usr/bin/tcpdump \
  && setcap 'cap_sys_ptrace+ep' /usr/bin/strace

RUN groupadd --gid $UID $USER \
  && useradd $USER --create-home --home-dir $HOME --no-log-init --uid $UID --gid $UID \
  && chown -R :root "${HOME}" && chmod -R g+s=u "${HOME}"

WORKDIR $HOME

ADD https://github.com/Nordix/ctraffic/releases/download/v1.7.0/ctraffic.gz ctraffic.gz
RUN gunzip ctraffic.gz \
  && chmod a+x ctraffic

ADD https://github.com/Nordix/mconnect/releases/download/v2.2.0/mconnect.xz mconnect.xz
RUN unxz mconnect.xz \
  && chmod a+x mconnect

COPY --from=build /app/examples/target/target-client .

USER ${UID}:${UID}
CMD ./ctraffic -server -address [::]:5000
