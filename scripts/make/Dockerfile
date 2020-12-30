# A docker file for scripts/make/build-docker.sh.

FROM alpine:3.12

ARG BUILD_DATE
ARG VERSION
ARG VCS_REF
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

# Update certificates.
RUN apk --no-cache --update add ca-certificates libcap && \
    rm -rf /var/cache/apk/* && \
    mkdir -p /opt/adguardhome/conf /opt/adguardhome/work && \
    chown -R nobody: /opt/adguardhome

ARG DIST_DIR
ARG TARGETARCH
ARG TARGETOS
ARG TARGETVARIANT

COPY --chown=nobody:nogroup\
	./${DIST_DIR}/docker/AdGuardHome_${TARGETOS}_${TARGETARCH}_${TARGETVARIANT}\
	/opt/adguardhome/AdGuardHome

RUN setcap 'cap_net_bind_service=+eip' /opt/adguardhome/AdGuardHome

EXPOSE 53/tcp 53/udp 67/udp 68/udp 80/tcp 443/tcp 853/tcp 3000/tcp

VOLUME ["/opt/adguardhome/conf", "/opt/adguardhome/work"]

WORKDIR /opt/adguardhome/work

ENTRYPOINT ["/opt/adguardhome/AdGuardHome"]

CMD [ \
	"--no-check-update", \
	"-c", "/opt/adguardhome/conf/AdGuardHome.yaml", \
	"-h", "0.0.0.0", \
	"-w", "/opt/adguardhome/work", \
]
