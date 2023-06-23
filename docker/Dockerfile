# A docker file for scripts/make/build-docker.sh.

FROM alpine:3.17

ARG BUILD_DATE
ARG VERSION
ARG VCS_REF

LABEL\
	maintainer="AdGuard Team <devteam@adguard.com>" \
	org.opencontainers.image.authors="AdGuard Team <devteam@adguard.com>" \
	org.opencontainers.image.created=$BUILD_DATE \
	org.opencontainers.image.description="Network-wide ads & trackers blocking DNS server" \
	org.opencontainers.image.documentation="https://github.com/AdguardTeam/AdGuardHome/wiki/" \
	org.opencontainers.image.licenses="GPL-3.0" \
	org.opencontainers.image.revision=$VCS_REF \
	org.opencontainers.image.source="https://github.com/AdguardTeam/AdGuardHome" \
	org.opencontainers.image.title="AdGuard Home" \
	org.opencontainers.image.url="https://adguard.com/en/adguard-home/overview.html" \
	org.opencontainers.image.vendor="AdGuard" \
	org.opencontainers.image.version=$VERSION

# Update certificates.
RUN apk --no-cache add ca-certificates libcap tzdata && \
	mkdir -p /opt/adguardhome/conf /opt/adguardhome/work && \
	chown -R nobody: /opt/adguardhome

RUN apk --no-cache add tini

ARG DIST_DIR
ARG TARGETARCH
ARG TARGETOS
ARG TARGETVARIANT

COPY --chown=nobody:nogroup\
	./${DIST_DIR}/docker/AdGuardHome_${TARGETOS}_${TARGETARCH}_${TARGETVARIANT}\
	/opt/adguardhome/AdGuardHome

RUN setcap 'cap_net_bind_service=+eip' /opt/adguardhome/AdGuardHome

# 53     : TCP, UDP : DNS
# 67     :      UDP : DHCP (server)
# 68     :      UDP : DHCP (client)
# 80     : TCP      : HTTP (main)
# 443    : TCP, UDP : HTTPS, DNS-over-HTTPS (incl. HTTP/3), DNSCrypt (main)
# 784    :      UDP : DNS-over-QUIC (experimental)
# 853    : TCP, UDP : DNS-over-TLS, DNS-over-QUIC
# 3000   : TCP, UDP : HTTP(S) (alt, incl. HTTP/3)
# 3001   : TCP, UDP : HTTP(S) (beta, incl. HTTP/3)
# 5443   : TCP, UDP : DNSCrypt (alt)
# 6060   : TCP      : HTTP (pprof)
# 8853   :      UDP : DNS-over-QUIC (experimental)
#
# TODO(a.garipov): Remove the old, non-standard 784 and 8853 ports for
# DNS-over-QUIC in a future release.
EXPOSE 53/tcp 53/udp 67/udp 68/udp 80/tcp 443/tcp 443/udp 784/udp\
	853/tcp 853/udp 3000/tcp 3000/udp 5443/tcp\
	5443/udp 6060/tcp 8853/udp

WORKDIR /opt/adguardhome/work

# Install helpers for healthcheck.
COPY --chown=nobody:nogroup\
	./${DIST_DIR}/docker/scripts\
	/opt/adguardhome/scripts

HEALTHCHECK \
	--interval=30s \
	--timeout=10s \
	--retries=3 \
	CMD [ "/opt/adguardhome/scripts/healthcheck.sh" ]

# It seems that the healthckech script sometimes spawns zombie processes, so we
# need a way to handle them, since AdGuard Home doesn't know how to keep track
# of the processes delegated to it by the OS.  Use tini as entry point because
# it needs the PID=1 to be the default parent for orphaned processes.
#
# See https://github.com/adguardTeam/adGuardHome/issues/3290.
ENTRYPOINT [ "/sbin/tini", "--" ]

CMD [ \
	"/opt/adguardhome/AdGuardHome", \
	"--no-check-update", \
	"-c", "/opt/adguardhome/conf/AdGuardHome.yaml", \
	"-w", "/opt/adguardhome/work" \
]
