# Keep the Makefile POSIX-compliant.  We currently allow hyphens in
# target names, but that may change in the future.
#
# See https://pubs.opengroup.org/onlinepubs/9699919799/utilities/make.html.
.POSIX:

CHANNEL = development
CLIENT_BETA_DIR = client2
CLIENT_DIR = client
COMMIT = $$(git rev-parse --short HEAD)
DIST_DIR = dist
GO = go
GOPROXY = https://goproxy.cn|https://proxy.golang.org|direct
GPG_KEY = devteam@adguard.com
GPG_KEY_PASSPHRASE = not-a-real-password
NPM = npm
NPM_FLAGS = --prefix $(CLIENT_DIR)
SIGN = 1
VERBOSE = 0
VERSION = v0.0.0
YARN = yarn
YARN_FLAGS = --cwd $(CLIENT_BETA_DIR) --network-timeout 120000

ENV = env\
	COMMIT='$(COMMIT)'\
	CHANNEL='$(CHANNEL)'\
	GPG_KEY='$(GPG_KEY)'\
	GPG_KEY_PASSPHRASE='$(GPG_KEY_PASSPHRASE)'\
	DIST_DIR='$(DIST_DIR)'\
	GO='$(GO)'\
	GOPROXY='$(GOPROXY)'\
	PATH="$${PWD}/bin:$$($(GO) env GOPATH)/bin:$${PATH}"\
	SIGN='$(SIGN)'\
	VERBOSE='$(VERBOSE)'\
	VERSION='$(VERSION)'\

# Keep the line above blank.

# Keep this target first, so that a naked make invocation triggers
# a full build.
build: deps quick-build

quick-build: js-build go-build

ci: deps test

deps: js-deps go-deps
lint: js-lint go-lint
test: js-test go-test

# Here and below, keep $(SHELL) in quotes, because on Windows this will
# expand to something like "C:/Program Files/Git/usr/bin/sh.exe".
build-docker: ; $(ENV) "$(SHELL)" ./scripts/make/build-docker.sh

build-release: deps js-build
	$(ENV) "$(SHELL)" ./scripts/make/build-release.sh

clean: ; $(ENV) "$(SHELL)" ./scripts/make/clean.sh
init:  ; git config core.hooksPath ./scripts/hooks

js-build:
	$(NPM) $(NPM_FLAGS) run build-prod
	$(YARN) $(YARN_FLAGS) build
js-deps:
	$(NPM) $(NPM_FLAGS) ci
	$(YARN) $(YARN_FLAGS) install

# TODO(a.garipov): Remove the legacy client tasks support once the new
# client is done and the old one is removed.
js-lint: ; $(NPM) $(NPM_FLAGS) run lint
js-test: ; $(NPM) $(NPM_FLAGS) run test
js-beta-lint: ; $(YARN) $(YARN_FLAGS) lint
js-beta-test: ; # TODO(v.abdulmyanov): Add tests for the new client.

go-build: ; $(ENV) "$(SHELL)" ./scripts/make/go-build.sh
go-deps:  ; $(ENV) "$(SHELL)" ./scripts/make/go-deps.sh
go-lint:  ; $(ENV) "$(SHELL)" ./scripts/make/go-lint.sh
go-test:  ; $(ENV) "$(SHELL)" ./scripts/make/go-test.sh
go-tools: ; $(ENV) "$(SHELL)" ./scripts/make/go-tools.sh

go-check: go-tools go-lint go-test

openapi-lint: ; cd ./openapi/ && $(YARN) test
openapi-show: ; cd ./openapi/ && $(YARN) start
