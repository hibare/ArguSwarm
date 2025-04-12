ARG GOLANG_VERSION=1.24
ARG VERSION=unknown

FROM golang:${GOLANG_VERSION}-alpine AS base

# ================== Build App ================== #

FROM base AS build

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

WORKDIR /src/

# Install healthcheck cmd
# hadolint ignore=DL3018
RUN apk update \
    && apk add --no-cache ca-certificates openssl \
    && apk add curl --no-cache \
    && apk add cosign --no-cache \
    && rm -rf /var/cache/apk/* \
    && update-ca-certificates \
    && curl -sfL https://raw.githubusercontent.com/hibare/go-docker-healthcheck/main/install.sh | sh -s -- -d -v -b /usr/local/bin

COPY . /src/

RUN go build -ldflags "-X github.com/hibare/ArguSwarm/internal/version.CurrentVersion=${VERSION}" -o arguswarm .

# ================== Final Image ================== #

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=build /usr/local/bin/healthcheck /bin/healthcheck

COPY --from=build /src/arguswarm /bin/arguswarm

EXPOSE 5000

ENTRYPOINT [ "arguswarm" ]
