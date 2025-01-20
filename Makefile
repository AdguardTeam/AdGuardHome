# Keep the Makefile POSIX-compliant.  We currently allow hyphens in
# target names, but that may change in the future.
#
# See https://pubs.opengroup.org/onlinepubs/9799919799/utilities/make.html.
.POSIX:

# This comment is used to simplify checking local copies of the
# Makefile.  Bump this number every time a significant change is made to
# this Makefile.
#
# AdGuard-Project-Version: 9

# Don't name these macros "GO" etc., because GNU Make apparently makes
# them exported environment variables with the literal value of
# "${GO:-go}" and so on, which is not what we need.  Use a dot in the
# name to make sure that users don't have an environment variable with
# the same name.
#
# See https://unix.stackexchange.com/q/646255/105635.
GO.MACRO = $${GO:-go}
VERBOSE.MACRO = $${VERBOSE:-0}

CHANNEL = development
CLIENT_DIR = client
DEPLOY_SCRIPT_PATH = not/a/real/path
DIST_DIR = dist
GOAMD64 = v1
GOPROXY = https://proxy.golang.org|direct
GOTELEMETRY = off
GOTOOLCHAIN = go1.23.5
GPG_KEY = devteam@adguard.com
GPG_KEY_PASSPHRASE = not-a-real-password
NPM = npm
NPM_FLAGS = --prefix $(CLIENT_DIR)
NPM_INSTALL_FLAGS = $(NPM_FLAGS) --quiet --no-progress --ignore-engines\
	--ignore-optional --ignore-platform --ignore-scripts
RACE = 0
REVISION = $${REVISION:-$$(git rev-parse --short HEAD)}
SIGN = 1
SIGNER_API_KEY = not-a-real-key
VERSION = v0.0.0
YARN = yarn

NEXTAPI = 0

# Macros for the build-release target.  If FRONTEND_PREBUILT is 0, the
# default, the macro $(BUILD_RELEASE_DEPS_$(FRONTEND_PREBUILT)) expands
# into BUILD_RELEASE_DEPS_0, and so both frontend and backend
# dependencies are fetched and the frontend is built.  Otherwise, if
# FRONTEND_PREBUILT is 1, only backend dependencies are fetched and the
# frontend isn't rebuilt.
#
# TODO(a.garipov): We could probably do that from .../build-release.sh,
# but that would mean either calling make from inside make or
# duplicating commands in two places, both of which don't seem to me
# like nice solutions.
FRONTEND_PREBUILT = 0
BUILD_RELEASE_DEPS_0 = deps js-build
BUILD_RELEASE_DEPS_1 = go-deps

ENV = env\
	CHANNEL='$(CHANNEL)'\
	DEPLOY_SCRIPT_PATH='$(DEPLOY_SCRIPT_PATH)' \
	DIST_DIR='$(DIST_DIR)'\
	GO="$(GO.MACRO)"\
	GOAMD64='$(GOAMD64)'\
	GOPROXY='$(GOPROXY)'\
	GOTELEMETRY='$(GOTELEMETRY)'\
	GOTOOLCHAIN='$(GOTOOLCHAIN)'\
	GPG_KEY='$(GPG_KEY)'\
	GPG_KEY_PASSPHRASE='$(GPG_KEY_PASSPHRASE)'\
	NEXTAPI='$(NEXTAPI)'\
	PATH="$${PWD}/bin:$$( "$(GO.MACRO)" env GOPATH )/bin:$${PATH}"\
	RACE='$(RACE)'\
	REVISION='$(REVISION)'\
	SIGN='$(SIGN)'\
	SIGNER_API_KEY='$(SIGNER_API_KEY)' \
	VERBOSE="$(VERBOSE.MACRO)"\
	VERSION="$(VERSION)"\

# Keep the line above blank.

ENV_MISC = env\
	PATH="$${PWD}/bin:$$("$(GO.MACRO)" env GOPATH)/bin:$${PATH}"\
	VERBOSE="$(VERBOSE.MACRO)"\

# Keep the line above blank.

# Keep this target first, so that a naked make invocation triggers a
# full build.
build: deps quick-build

init: ; git config core.hooksPath ./scripts/hooks

quick-build: js-build go-build

deps: js-deps go-deps
lint: js-lint go-lint
test: js-test go-test

# Here and below, keep $(SHELL) in quotes, because on Windows this will
# expand to something like "C:/Program Files/Git/usr/bin/sh.exe".
build-docker: ; $(ENV) "$(SHELL)" ./scripts/make/build-docker.sh

build-release: $(BUILD_RELEASE_DEPS_$(FRONTEND_PREBUILT))
	$(ENV) "$(SHELL)" ./scripts/make/build-release.sh

js-build: ; $(NPM) $(NPM_FLAGS) run build-prod
js-deps:  ; $(NPM) $(NPM_INSTALL_FLAGS) ci
js-lint:  ; $(NPM) $(NPM_FLAGS) run lint
js-test:  ; $(NPM) $(NPM_FLAGS) run test

go-bench:     ; $(ENV) "$(SHELL)"    ./scripts/make/go-bench.sh
go-build:     ; $(ENV) "$(SHELL)"    ./scripts/make/go-build.sh
go-deps:      ; $(ENV) "$(SHELL)"    ./scripts/make/go-deps.sh
go-env:       ; $(ENV) "$(GO.MACRO)" env
go-fuzz:      ; $(ENV) "$(SHELL)"    ./scripts/make/go-fuzz.sh
go-lint:      ; $(ENV) "$(SHELL)"    ./scripts/make/go-lint.sh
# TODO(a.garipov): Think about making RACE='1' the default for all
# targets.
go-test:      ; $(ENV) RACE='1' "$(SHELL)" ./scripts/make/go-test.sh
go-tools:     ; $(ENV)          "$(SHELL)" ./scripts/make/go-tools.sh
go-upd-tools: ; $(ENV)          "$(SHELL)" ./scripts/make/go-upd-tools.sh

go-check: go-tools go-lint go-test

# A quick check to make sure that all operating systems relevant to the
# development of the project can be typechecked and built successfully.
go-os-check:
	$(ENV) GOOS='darwin'  "$(GO.MACRO)" vet ./internal/...
	$(ENV) GOOS='freebsd' "$(GO.MACRO)" vet ./internal/...
	$(ENV) GOOS='openbsd' "$(GO.MACRO)" vet ./internal/...
	$(ENV) GOOS='linux'   "$(GO.MACRO)" vet ./internal/...
	$(ENV) GOOS='windows' "$(GO.MACRO)" vet ./internal/...

txt-lint: ; $(ENV) "$(SHELL)" ./scripts/make/txt-lint.sh

md-lint:  ; $(ENV_MISC) "$(SHELL)" ./scripts/make/md-lint.sh
sh-lint:  ; $(ENV_MISC) "$(SHELL)" ./scripts/make/sh-lint.sh

openapi-lint: ; cd ./openapi/ && $(YARN) test
openapi-show: ; cd ./openapi/ && $(YARN) start
