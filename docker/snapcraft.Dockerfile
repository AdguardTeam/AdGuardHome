# syntax=docker/dockerfile:1

# This comment is used to simplify checking local copies of the Dockerfile.
# Bump this number every time a significant change is made to this Dockerfile.
#
# AdGuard-Project-Version: 11

# Dockerfile guidelines:
#
# 1. Make sure that Docker correctly caches layers, on a second build attempt it
#    must not run lint / test second time when it's not required.
#
# 2. Use BuildKit to improve the build performance (--mount=type=cache, etc).
#
# 3. Prefer using ARG instead of ENV when appropriate, as ARG does not create a
#    layer in the final image.  However, be careful with what you use ARG for.
#    Also, prefer to give ARGs sensible default values.
#
# 4. Use --output and the export stage if you need to get any output on the host
#    machine.
#
#    NOTE:  Only use --output with FROM scratch.
#
# 5. Use .dockerignore to prevent unnecessary files from being sent to the
#    Docker daemon, which can invalidate the cache.
#
# 6. Add a CACHE_BUSTER argument to stages to be able to rerun the stages if
#    needed.  Keep it in sync with bamboo-specs/snapcraft.yaml.

# NOTE:  Keep in sync with bamboo-specs/snapcraft.yaml.
ARG BASE_IMAGE=adguard/snap-builder:2.1

# builder downloads the release artifacts and builds snap artifacts.
FROM "$BASE_IMAGE" AS builder
ARG CACHE_BUSTER=0
ARG CHANNEL=development
ARG VERSION=""
ADD snap /app/snap
ADD scripts /app/scripts
WORKDIR /app
RUN \
<<-'EOF'
set -e -f -u -x

export VERBOSE='1'

env \
	CHANNEL="${CHANNEL}" \
	sh ./scripts/snap/download.sh \
	;

sh ./scripts/snap/build.sh
EOF

# builder-exporter exports the build artifacts to the host machine so that they
# could be published.  This stage should only be used in a CI.
FROM scratch AS builder-exporter
ARG CACHE_BUSTER=0
ARG VERSION=""
COPY --from=builder /app/AdGuardHome_amd64.snap /AdGuardHome_amd64.snap
COPY --from=builder /app/AdGuardHome_arm64.snap /AdGuardHome_arm64.snap
COPY --from=builder /app/AdGuardHome_armhf.snap /AdGuardHome_armhf.snap
COPY --from=builder /app/AdGuardHome_i386.snap /AdGuardHome_i386.snap

# publisher uploads the release artifacts to the Snap Store.
FROM "$BASE_IMAGE" AS publisher
ARG CACHE_BUSTER=0
ARG SNAPCRAFT_CHANNEL=0
ARG SNAPCRAFT_STORE_CREDENTIALS=0
ARG VERSION=""
ADD snap /app/snap
ADD scripts /app/scripts
ADD AdGuardHome_amd64.snap /app/AdGuardHome_amd64.snap
ADD AdGuardHome_arm64.snap /app/AdGuardHome_arm64.snap
ADD AdGuardHome_armhf.snap /app/AdGuardHome_armhf.snap
ADD AdGuardHome_i386.snap /app/AdGuardHome_i386.snap
WORKDIR /app
RUN \
<<-'EOF'
set -e -f -u -x

env \
	SNAPCRAFT_CHANNEL="${SNAPCRAFT_CHANNEL}" \
	SNAPCRAFT_STORE_CREDENTIALS="${SNAPCRAFT_STORE_CREDENTIALS}" \
	VERBOSE='1' \
	sh ./scripts/snap/upload.sh
EOF
