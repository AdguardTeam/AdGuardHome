FROM alpine:latest
LABEL maintainer="Erik Rogers <erik.rogers@live.com>"

# AdGuard version
ARG ADGUARD_VERSION="0.9"
ENV ADGUARD_VERSION $ADGUARD_VERSION

# AdGuard architecture and package info
ARG ADGUARD_ARCH="linux_amd64"
ENV ADGUARD_ARCH ${ADGUARD_ARCH}
ENV ADGUARD_PACKAGE "AdGuardHome_v${ADGUARD_VERSION}_${ADGUARD_ARCH}"

# AdGuard release info
ARG ADGUARD_ARCHIVE="${ADGUARD_PACKAGE}.tar.gz"
ENV ADGUARD_ARCHIVE ${ADGUARD_ARCHIVE}
ARG ADGUARD_RELEASE="https://github.com/AdguardTeam/AdGuardHome/releases/download/v${ADGUARD_VERSION}/${ADGUARD_ARCHIVE}"
ENV ADGUARD_RELEASE ${ADGUARD_RELEASE}

# AdGuard directory
ARG ADGUARD_DIR="/data/adguard"
ENV ADGUARD_DIR ${ADGUARD_DIR}

# Update CA certs and download AdGuard binaries
RUN apk --no-cache --update add ca-certificates \
  && cd /tmp \
  && wget ${ADGUARD_RELEASE} \
  && tar xvf ${ADGUARD_ARCHIVE} \
  && mkdir -p "${ADGUARD_DIR}" \
  && cp "AdGuardHome/AdGuardHome" "${ADGUARD_DIR}" \
  && chmod +x "${ADGUARD_DIR}/AdGuardHome" \
  && rm -rf "AdGuardHome" \
  && rm ${ADGUARD_ARCHIVE}

# Expose DNS port 53
EXPOSE 53

# Expose UI port 3000
ARG ADGUARD_UI_HOST="0.0.0.0"
ENV ADGUARD_UI_HOST ${ADGUARD_UI_HOST}
ARG ADGUARD_UI_PORT="3000"
ENV ADGUARD_UI_PORT ${ADGUARD_UI_PORT}

EXPOSE ${ADGUARD_UI_PORT}

# Run AdGuardHome
WORKDIR ${ADGUARD_DIR}
VOLUME ${ADGUARD_DIR}
ENTRYPOINT ./AdGuardHome --host ${ADGUARD_UI_HOST} --port ${ADGUARD_UI_PORT}
