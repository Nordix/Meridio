ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest
ARG USER=meridio
ARG UID=10005
ARG HOME=/home/${USER}

FROM golang:1.23 as build
ARG LDFLAGS

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -ldflags="${LDFLAGS}" -a -o manager ./cmd/operator/main.go

FROM ${base_image}
ARG USER
ARG UID
ARG HOME
RUN addgroup --gid $UID $USER \
  && adduser $USER --home $HOME --uid $UID -G $USER --disabled-password \
  && chown -R :root "${HOME}" && chmod -R g+s=u "${HOME}"
WORKDIR $HOME
COPY --from=build /workspace/manager .
USER ${UID}:${UID}
CMD ["./manager"]
