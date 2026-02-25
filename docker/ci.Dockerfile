# syntax=docker/dockerfile:1

# This comment is used to simplify checking local copies of the Dockerfile.
# Bump this number every time a significant change is made to this Dockerfile.
#
# AdGuard-Project-Version: 10

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
ARG BASE_IMAGE=adguard/go-builder:1.25.7--1

# The dependencies stage is needed to install packages and tool dependencies.
# This is also where binaries like osslsigncode, which may be required for tests
# in some projects, must be installed.
#
# Use fake BRANCH and REVISION values to both prevent git calls and also not
# ruin the caching with ARGs.
#
# NOTE:  Only ADD the files required to install the dependencies.
FROM "$BASE_IMAGE" AS dependencies
ADD Makefile go.mod go.sum /app/
ADD scripts /app/scripts
WORKDIR /app
RUN \
	--mount=type=cache,id=gocache,target=/root/.cache/go-build \
	--mount=type=cache,id=gopath,target=/go \
<<-'EOF'
set -e -f -u -x
make \
	BRANCH='master' \
	REVISION='0000000000000000000000000000000000000000' \
	VERBOSE=1 \
	go-env \
	go-deps \
	;
EOF

# The linter stage is separated from the tester stage to make catching test
# failures easier.
#
# Use fake BRANCH and REVISION values to both prevent git calls and also not
# ruin the caching with ARGs.  IGNORE_NON_REPRODUCIBLE is set to 1 to make this
# stage reproducible even when linters that query external sources fail.
FROM dependencies AS linter
ADD . /app
WORKDIR /app
RUN \
	--mount=type=cache,id=gocache,target=/root/.cache/go-build \
	--mount=type=cache,id=gopath,target=/go \
<<-'EOF'
set -e -f -u -x
export GOMAXPROCS=2
make \
	BRANCH='master' \
	IGNORE_NON_REPRODUCIBLE='1' \
	REVISION='0000000000000000000000000000000000000000' \
	VERBOSE=1 \
	go-lint \
	md-lint \
	sh-lint \
	txt-lint \
	;
EOF

# The test stage.  TEST_REPORTS_DIR is set to create JUnit reports for the
# tester-exporter stage; run with --build-arg TEST_REPORTS_DIR='' if you don't
# need them on your machine.
#
# Use fake BRANCH and REVISION values to both prevent git calls and also not
# ruin the caching with ARGs.
#
# To run the tests:
#
#   docker build --target tester -t 'app' .
#
# Projects that have go-bench and/or go-fuzz targets should add them here as
# well.
FROM linter AS tester
ARG CACHE_BUSTER=0
ARG TEST_REPORTS_DIR=/test-reports
RUN \
	--mount=type=cache,id=gocache,target=/root/.cache/go-build \
	--mount=type=cache,id=gopath,target=/go \
<<-'EOF'
set -e -f -u -x
export GOMAXPROCS=2

make \
	BRANCH='master' \
	REVISION='0000000000000000000000000000000000000000' \
	TEST_REPORTS_DIR="$TEST_REPORTS_DIR" \
	VERBOSE=1 \
	go-test \
	;

exit_code="$(cat "${TEST_REPORTS_DIR}/test-exit-code.txt")"
readonly exit_code

make \
	BRANCH='master' \
	REVISION='0000000000000000000000000000000000000000' \
	VERBOSE=1 \
	go-fuzz \
	go-bench \
	;

exit "$exit_code"
EOF

# tester-exporter exports the test result to the host machine so that it could
# parse and analyze it.  This stage should only used in a CI.
#
# It the file test-report.xml, which contains test results in the JUnit format.
#
# Run the following command to export the test result:
#
#   docker build \
#	   --output . \
#	   --progress plain \
#	   --target tester-exporter \
#	   .
FROM scratch AS tester-exporter
ARG CACHE_BUSTER=0
ARG TEST_REPORTS_DIR=/test-reports
COPY --from=tester "$TEST_REPORTS_DIR" "$TEST_REPORTS_DIR"

# The builder stage is used to build release artifacts.  Real BRANCH and
# REVISION must be used here.
FROM dependencies AS builder
ARG ARCH=""
ARG BRANCH=master
ARG CACHE_BUSTER=0
ARG CHANNEL=development
ARG DEPLOY_SCRIPT_PATH=not/a/real/path
ARG GPG_KEY_PASSPHRASE
ARG GPG_SECRET_KEY
ARG OS=""
ARG REVISION=0000000000000000000000000000000000000000
ARG SIGNER_API_KEY
ARG SOURCE_DATE_EPOCH=0
ARG VERSION=""
ADD . /app
WORKDIR /app
RUN \
	--mount=type=cache,id=gocache,target=/root/.cache/go-build \
	--mount=type=cache,id=gopath,target=/go \
<<-'EOF'
set -e -f -u -x

make \
	ARCH="${ARCH}" \
	BRANCH="${BRANCH}" \
	CHANNEL="${CHANNEL}" \
	FRONTEND_PREBUILT=1 \
	OS="${OS}" \
	PARALLELISM=1 \
	REVISION="${REVISION}" \
	SOURCE_DATE_EPOCH="$SOURCE_DATE_EPOCH" \
	SIGN=0 \
	VERBOSE=2 \
	VERSION="${VERSION}" \
	build-release \
	;
EOF

# builder-exporter exports the build artifacts to the host machine so that they
# could be published.  This stage should only be used in a CI.
FROM scratch AS builder-exporter
ARG CACHE_BUSTER=0
COPY --from=builder /app/dist /dist
