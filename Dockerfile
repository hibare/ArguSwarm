# syntax=docker/dockerfile:1.7

ARG GOLANG_VERSION=1.27.3

ARG VERSION=unknown
ARG BUILD_TIMESTAMP=unknown
ARG COMMIT_HASH=unknown

# =============================================================================
# Backend
# =============================================================================

FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION}-alpine AS backend-build

ARG TARGETOS
ARG TARGETARCH

ARG VERSION
ARG BUILD_TIMESTAMP
ARG COMMIT_HASH

WORKDIR /src

# hadolint ignore=DL3064
ENV GOFLAGS=-mod=readonly

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

# Build dependencies only.
# hadolint ignore=DL3018
RUN apk add --no-cache \
        ca-certificates \
        curl \
        git \
    && update-ca-certificates

# Keep Go dependency resolution independently cacheable.
COPY go.mod go.sum ./

RUN --mount=type=cache,id=arguswarm-go-mod-${TARGETARCH},target=/go/pkg/mod \
    go mod download

# Application source.
COPY . .

# Cross-compile on BUILDPLATFORM for TARGETOS/TARGETARCH.
RUN --mount=type=cache,id=arguswarm-go-mod-${TARGETARCH},target=/go/pkg/mod \
    --mount=type=cache,id=arguswarm-go-build-${TARGETARCH},target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS="${TARGETOS}" \
    GOARCH="${TARGETARCH}" \
    go build \
        -mod=readonly \
        -trimpath \
        -ldflags="-s -w \
          -X github.com/hibare/ArguSwarm/internal/constants.Version=${VERSION} \
          -X github.com/hibare/ArguSwarm/internal/constants.BuildTimestamp=${BUILD_TIMESTAMP} \
          -X github.com/hibare/ArguSwarm/internal/constants.CommitHash=${COMMIT_HASH}" \
        -o /src/arguswarm ./main.go

# =============================================================================
# Runtime
# =============================================================================

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Runtime dependencies only.
# hadolint ignore=DL3018
RUN apk add --no-cache \
        ca-certificates \
        curl \
    && update-ca-certificates

COPY --from=backend-build /src/arguswarm /bin/arguswarm

EXPOSE 8080 8081

ENTRYPOINT ["/bin/arguswarm"]
