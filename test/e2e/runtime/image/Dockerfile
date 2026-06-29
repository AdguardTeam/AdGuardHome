FROM debian:bookworm-slim

ARG ADGUARD_VERSION=v0.107.52
ARG ADGUARD_ARCHIVE=AdGuardHome_linux_amd64.tar.gz
ARG GODNSBENCH_VERSION=v1.10.0
ARG DNSLOOKUP_VERSION=v1.11.1

RUN apt-get update && apt-get install -y --no-install-recommends \
      curl ca-certificates procps busybox openssl \
    && rm -rf /var/lib/apt/lists/*

# In-container DNS tooling: queries run against 127.0.0.1:53 inside the
# container, so DNS tests never depend on host UDP port mapping.
RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "${arch}" in amd64|arm64) a="${arch}" ;; *) echo "unsupported: ${arch}" >&2; exit 1 ;; esac; \
    curl -fsSL "https://github.com/ameshkov/godnsbench/releases/download/${GODNSBENCH_VERSION}/godnsbench-linux-${a}-${GODNSBENCH_VERSION}.tar.gz" \
      | tar -xz --strip-components=1 -C /usr/local/bin "linux-${a}/godnsbench"; \
    curl -fsSL "https://github.com/ameshkov/dnslookup/releases/download/${DNSLOOKUP_VERSION}/dnslookup-linux-${a}-${DNSLOOKUP_VERSION}.tar.gz" \
      | tar -xz --strip-components=1 -C /usr/local/bin "linux-${a}/dnslookup"; \
    chmod +x /usr/local/bin/godnsbench /usr/local/bin/dnslookup

# Prefer a build-context tarball (CI caches it); fall back to pinned download.
COPY runtime/image/install-agh.sh /tmp/install-agh.sh
COPY ${ADGUARD_ARCHIVE}* /tmp/
RUN chmod +x /tmp/install-agh.sh && /tmp/install-agh.sh "${ADGUARD_VERSION}"

RUN mkdir -p /opt/adguardhome/conf /opt/adguardhome/work /pristine
RUN cat > /pristine/AdGuardHome.yaml <<'EOF'
http:
  address: 0.0.0.0:3000
dns:
  bind_hosts:
    - 0.0.0.0
  port: 53
  safesearch_enabled: false
users:
  - name: admin
    password: $2b$12$aw6lk4Cdfc/b69rFQVqSrutVmh6UJ.ORxpQ10.fj685NVWmDiDO9O
# Ship with NO subscription filters: AGH otherwise injects its default lists
# (e.g. the 157k-rule "AdGuard DNS filter") which makes /control/filtering/refresh
# re-download ~3.7MB from the public internet on every call — slow and a hidden
# egress dependency. Tests add their own local blocklists as needed.
filters: []
whitelist_filters: []
schema_version: 20
EOF
COPY runtime/image/agh-entrypoint.sh /usr/local/bin/agh-entrypoint.sh
COPY runtime/image/agh-reset.sh /usr/local/bin/agh-reset.sh
COPY runtime/image/agh-apply.sh /usr/local/bin/agh-apply.sh
RUN chmod +x /usr/local/bin/agh-entrypoint.sh /usr/local/bin/agh-reset.sh /usr/local/bin/agh-apply.sh

EXPOSE 3000 53/tcp 53/udp
ENTRYPOINT ["/usr/local/bin/agh-entrypoint.sh"]
