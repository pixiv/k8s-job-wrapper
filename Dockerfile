ARG GO_VERSION=1.25
# Build the manager binary
FROM golang:$GO_VERSION AS builder
ARG KUBECTL_VERSION=v1.35.0
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Download kubectl
RUN --mount=type=bind,source=hack,target=. \
    ./setup-kubectl.sh $KUBECTL_VERSION /usr/local/bin/kubectl
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/go/pkg/mod,id=gomodcache \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download
# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN --mount=type=cache,target=/go/pkg/mod,id=gomodcache \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o /usr/local/bin/manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /usr/local/bin/manager /usr/local/bin/kubectl /
USER 65532:65532

ENTRYPOINT ["/manager"]
