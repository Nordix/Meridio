ARG base_image=registry.nordix.org/cloud-native/meridio/base:latest
ARG USER=meridio
ARG UID=10004
ARG HOME=/home/${USER}

FROM golang:1.23 as build

ENV GO111MODULE=on
ARG meridio_version=0.0.0-unknown

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-extldflags -static -X main.version=${meridio_version}" -o ipam ./cmd/ipam

FROM ${base_image}
ARG USER
ARG UID
ARG HOME
RUN addgroup --gid $UID $USER \
  && adduser $USER --home $HOME --uid $UID -G $USER --disabled-password \
  && chown -R :root "${HOME}" && chmod -R g+s=u "${HOME}"
WORKDIR $HOME
COPY --from=build /app/ipam .
# note: To run as non-root user, cap_dac_override file capability might be required in case hostPath
# volumes are used where the mounted contents file permissions do not allow "others" the required
# access modes (or permissions weren't adjusted by some other means).
# For example, hostPath unix spire-agent-socket's file permissions grant "write" access for "others", thus
# cap_dac_override is not needed by the nsp binary.
# Similarly in development environments (e.g. Kind, xcluster) in case of Rancher's Local Path based
# persistent storage the mounted directory's file permissions will allow 'rwx' access for "others".
# Yet, if the persistent storage file already exists when for example upgrading from a root user
# deployment (or simply the user id changes), access problems might arise in case of hostPath based
# persistent storage solutions. Otherwise e.g. "fsGroup" should secure access through group ownership.
#RUN setcap 'cap_dac_override+ep' ./ipam
USER ${UID}:${UID}
CMD ["./ipam"]
