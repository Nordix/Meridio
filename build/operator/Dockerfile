ARG BUILDER=golang:1.19.0
ARG BASE_IMG=ubuntu:22.04

# Build the manager binary
FROM ${BUILDER} as builder
ARG LDFLAGS

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -ldflags="${LDFLAGS}" -a -o manager ./cmd/operator/main.go

# Copy files from builder to the final image
FROM ${BASE_IMG}
WORKDIR /
COPY --from=builder /workspace/manager .

USER 65532:65532

ENTRYPOINT ["/manager"]
