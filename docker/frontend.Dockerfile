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
#    needed.  Keep it in sync with bamboo-specs/bamboo.yaml.

# NOTE:  Keep in sync with bamboo-specs/bamboo.yaml.
ARG BASE_IMAGE=adguard/home-js-builder:4.0

# The dependencies stage is needed to install packages and tool dependencies.
# This is also where binaries like osslsigncode, which may be required for tests
# in some projects, must be installed.
#
# NOTE:  Only ADD the files required to install the dependencies.
FROM "$BASE_IMAGE" AS dependencies
ADD Makefile /app/
ADD scripts /app/scripts
ADD client /app/client
WORKDIR /app
RUN \
    --mount=type=cache,id=npm-root-cache,target=/root/.npm \
<<-'EOF'
set -e -f -u -x
make \
	VERBOSE=1 \
	js-deps \
	;
EOF

# The linter stage is separated from the tester stage to make catching test
# failures easier.
FROM dependencies AS linter
ARG CACHE_BUSTER=0
ADD . /app
WORKDIR /app
RUN \
    --mount=type=cache,id=npm-root-cache,target=/root/.npm \
<<-'EOF'
set -e -f -u -x
make \
	VERBOSE=1 \
	js-typecheck \
	js-lint \
	;
EOF

# The test stage.
FROM linter AS tester
ARG CACHE_BUSTER=0
RUN \
    --mount=type=cache,id=npm-root-cache,target=/root/.npm \
<<-'EOF'
set -e -f -u -x
make \
	VERBOSE=1 \
	js-test \
	;
EOF

# The e2e test stage.
FROM dependencies AS e2etester
ARG CACHE_BUSTER=0
ADD . /app
WORKDIR /app
RUN \
    --mount=type=cache,id=npm-root-cache,target=/root/.npm \
<<-'EOF'
set -e -f -u -x
make \
	CI='true' \
	VERBOSE=1 \
	js-test-e2e \
	;
EOF

# The builder stage.
FROM dependencies AS builder
ARG CACHE_BUSTER=0
ADD . /app
WORKDIR /app
RUN \
    --mount=type=cache,id=npm-root-cache,target=/root/.npm \
<<-'EOF'
set -e -f -u -x
make \
	VERBOSE=1 \
	js-build \
	;
EOF

# builder-exporter exports the build artifacts to the host machine so that they
# could be published.  This stage should only be used in a CI.
FROM scratch AS builder-exporter
ARG CACHE_BUSTER=0
COPY --from=builder /app/build /build
