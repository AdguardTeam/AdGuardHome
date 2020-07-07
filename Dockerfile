FROM --platform=${BUILDPLATFORM:-linux/amd64} tonistiigi/xx:golang AS xgo
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.14-alpine as builder

ARG BUILD_DATE
ARG VCS_REF
ARG VERSION=dev
ARG CHANNEL=release

ENV CGO_ENABLED 0
ENV GO111MODULE on
ENV GOPROXY https://goproxy.io

COPY --from=xgo / /
RUN go env

RUN apk --update --no-cache add \
    build-base \
    gcc \
    git \
    npm \
  && rm -rf /tmp/* /var/cache/apk/*

WORKDIR /app

COPY . ./

# Prepare the client code
RUN npm --prefix client ci && npm --prefix client run build-prod

# Download go dependencies
RUN go mod download
RUN go generate ./...

# It's important to place TARGET* arguments here to avoid running npm and go mod download for every platform
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
RUN go build -ldflags="-s -w -X main.version=${VERSION} -X main.channel=${CHANNEL} -X main.goarm=${GOARM}"

FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:latest

ARG BUILD_DATE
ARG VCS_REF
ARG VERSION
ARG CHANNEL

LABEL maintainer="AdGuard Team <devteam@adguard.com>" \
  org.opencontainers.image.created=$BUILD_DATE \
  org.opencontainers.image.url="https://adguard.com/adguard-home.html" \
  org.opencontainers.image.source="https://github.com/AdguardTeam/AdGuardHome" \
  org.opencontainers.image.version=$VERSION \
  org.opencontainers.image.revision=$VCS_REF \
  org.opencontainers.image.vendor="AdGuard" \
  org.opencontainers.image.title="AdGuard Home" \
  org.opencontainers.image.description="Network-wide ads & trackers blocking DNS server" \
  org.opencontainers.image.licenses="GPL-3.0"

RUN apk --update --no-cache add \
    ca-certificates \
    libcap \
    libressl \
    tzdata \
  && rm -rf /tmp/* /var/cache/apk/*

COPY --from=builder --chown=nobody:nogroup /app/AdGuardHome /opt/adguardhome/AdGuardHome
COPY --from=builder --chown=nobody:nogroup /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip

RUN /opt/adguardhome/AdGuardHome --version \
  && mkdir -p /opt/adguardhome/conf /opt/adguardhome/work \
  && chown -R nobody: /opt/adguardhome \
  && setcap 'cap_net_bind_service=+eip' /opt/adguardhome/AdGuardHome

EXPOSE 53/tcp 53/udp 67/udp 68/udp 80/tcp 443/tcp 853/tcp 3000/tcp
WORKDIR /opt/adguardhome/work
VOLUME ["/opt/adguardhome/conf", "/opt/adguardhome/work"]

ENTRYPOINT ["/opt/adguardhome/AdGuardHome"]
CMD ["-h", "0.0.0.0", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work", "--no-check-update"]
