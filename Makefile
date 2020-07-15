#
# Available targets
#
# * build -- builds AdGuardHome for the current platform
# * client -- builds client-side code of AdGuard Home
# * client-watch -- builds client-side code of AdGuard Home and watches for changes there
# * docker -- builds a docker image for the current platform
# * clean -- clean everything created by previous builds
# * lint -- run all linters
# * test -- run all unit-tests
# * dependencies -- installs dependencies (go and npm modules)
# * ci -- installs dependencies, runs linters and tests, intended to be used by CI/CD
#
# Building releases:
#
# * release -- builds AdGuard Home distros. CHANNEL must be specified (edge, release or beta).
# * docker-multi-arch -- builds a multi-arch image. If you want it to be pushed to docker hub,
# 	you must specify:
#     * DOCKER_IMAGE_NAME - adguard/adguard-home
#     * DOCKER_OUTPUT - type=image,name=adguard/adguard-home,push=true

GOPATH := $(shell go env GOPATH)
PWD := $(shell pwd)
TARGET=AdGuardHome
BASE_URL="https://static.adguard.com/adguardhome/$(CHANNEL)"

# See release target
DIST_DIR=dist

# Update channel. Can be release, beta or edge. Uses edge by default.
CHANNEL ?= edge

# Validate channel
ifneq ($(CHANNEL),release)
ifneq ($(CHANNEL),beta)
ifneq ($(CHANNEL),edge)
$(error CHANNEL value is not valid. Valid values are release,beta or edge)
endif
endif
endif

# goreleaser command depends on the $CHANNEL
GORELEASER_COMMAND=goreleaser release --rm-dist --skip-publish --snapshot
ifneq ($(CHANNEL),edge)
	# If this is not an "edge" build, use normal release command
	GORELEASER_COMMAND=goreleaser release --rm-dist --skip-publish
endif

# Version properties
COMMIT=$(shell git rev-parse --short HEAD)
TAG_NAME=$(shell git describe --abbrev=0)
RELEASE_VERSION=$(TAG_NAME)
SNAPSHOT_VERSION=$(RELEASE_VERSION)-SNAPSHOT-$(COMMIT)

# Set proper version
VERSION=
ifeq ($(TAG_NAME),$(shell git describe --abbrev=4))
	VERSION=$(RELEASE_VERSION)
else
	VERSION=$(SNAPSHOT_VERSION)
endif

# Docker target parameters
DOCKER_IMAGE_NAME ?= adguardhome-dev
DOCKER_IMAGE_FULL_NAME = $(DOCKER_IMAGE_NAME):$(VERSION)
DOCKER_PLATFORMS=linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64,linux/386,linux/ppc64le
DOCKER_OUTPUT ?= type=image,name=$(DOCKER_IMAGE_NAME),push=false
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Docker tags (can be redefined)
DOCKER_TAGS ?=
ifndef DOCKER_TAGS
	ifeq ($(CHANNEL),release)
		DOCKER_TAGS := --tag $(DOCKER_IMAGE_NAME):latest
	endif
	ifeq ($(CHANNEL),beta)
		DOCKER_TAGS := --tag $(DOCKER_IMAGE_NAME):beta
	endif
	ifeq ($(CHANNEL),edge)
		# Don't set the version tag when pushing to "edge"
		DOCKER_IMAGE_FULL_NAME := $(DOCKER_IMAGE_NAME):edge
		# DOCKER_TAGS := --tag $(DOCKER_IMAGE_NAME):edge
	endif
endif

# Validate docker build arguments
ifndef DOCKER_IMAGE_NAME
$(error DOCKER_IMAGE_NAME value is not set)
endif

.PHONY: all build client client-watch docker lint lint-js lint-go test dependencies clean release docker-multi-arch
all: build

init:
	git config core.hooksPath .githooks

build: dependencies client
	PATH=$(GOPATH)/bin:$(PATH) go generate ./...
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION) -X main.channel=$(CHANNEL) -X main.goarm=$(GOARM)"
	PATH=$(GOPATH)/bin:$(PATH) packr clean

client:
	npm --prefix client run build-prod

client-watch:
	npm --prefix client run watch

docker:
	DOCKER_CLI_EXPERIMENTAL=enabled \
	docker buildx build \
	--build-arg VERSION=$(VERSION) \
	--build-arg CHANNEL=$(CHANNEL) \
	--build-arg VCS_REF=$(COMMIT) \
	--build-arg BUILD_DATE=$(BUILD_DATE) \
	$(DOCKER_TAGS) \
	--load \
	-t "$(DOCKER_IMAGE_NAME)" -f ./Dockerfile .

	@echo Now you can run the docker image:
	@echo docker run --name "adguard-home" -p 53:53/tcp -p 53:53/udp -p 80:80/tcp -p 443:443/tcp -p 853:853/tcp -p 3000:3000/tcp $(DOCKER_IMAGE_NAME)

lint: lint-js lint-go

lint-js:
	@echo Running js linter
	npm --prefix client run lint

lint-go:
	@echo Running go linter
	golangci-lint run

test:
	@echo Running unit-tests
	go test -race -v -bench=. -coverprofile=coverage.txt -covermode=atomic ./...

ci: dependencies client test

dependencies:
	npm --prefix client ci
	go mod download

clean:
	# make build output
	rm -f AdGuardHome
	rm -f AdGuardHome.exe
	# tests output
	rm -rf data
	rm -f coverage.txt
	# static build output
	rm -rf build
	# dist folder
	rm -rf $(DIST_DIR)
	# client deps
	rm -rf client/node_modules
	# packr-generated files
	PATH=$(GOPATH)/bin:$(PATH) packr clean || true

docker-multi-arch:
	DOCKER_CLI_EXPERIMENTAL=enabled \
	docker buildx build \
	--platform $(DOCKER_PLATFORMS) \
	--build-arg VERSION=$(VERSION) \
	--build-arg CHANNEL=$(CHANNEL) \
	--build-arg VCS_REF=$(COMMIT) \
	--build-arg BUILD_DATE=$(BUILD_DATE) \
	$(DOCKER_TAGS) \
	--output "$(DOCKER_OUTPUT)" \
	-t "$(DOCKER_IMAGE_FULL_NAME)" -f ./Dockerfile .

	@echo If the image was pushed to the registry, you can now run it:
	@echo docker run --name "adguard-home" -p 53:53/tcp -p 53:53/udp -p 80:80/tcp -p 443:443/tcp -p 853:853/tcp -p 3000:3000/tcp $(DOCKER_IMAGE_NAME)

release: dependencies client
	@echo Starting release build: version $(VERSION), channel $(CHANNEL)
	CHANNEL=$(CHANNEL) $(GORELEASER_COMMAND)
	$(call write_version_file,$(VERSION))
	PATH=$(GOPATH)/bin:$(PATH) packr clean

define write_version_file
	$(eval version := $(1))

	@echo Writing version file: $(version)

	# Variables for CI
	rm -f $(DIST_DIR)/version.txt
	echo "version=$(version)" > $(DIST_DIR)/version.txt

	# Prepare the version.json file
	rm -f $(DIST_DIR)/version.json
	echo "{" >> $(DIST_DIR)/version.json
	echo "  \"version\": \"$(version)\"," >> $(DIST_DIR)/version.json
	echo "  \"announcement\": \"AdGuard Home $(version) is now available!\"," >> $(DIST_DIR)/version.json
	echo "  \"announcement_url\": \"https://github.com/AdguardTeam/AdGuardHome/releases\"," >> $(DIST_DIR)/version.json
	echo "  \"selfupdate_min_version\": \"v0.0\"," >> $(DIST_DIR)/version.json

	# Windows builds
	echo "  \"download_windows_amd64\": \"$(BASE_URL)/AdGuardHome_windows_amd64.zip\"," >> $(DIST_DIR)/version.json
	echo "  \"download_windows_386\": \"$(BASE_URL)/AdGuardHome_windows_386.zip\"," >> $(DIST_DIR)/version.json

	# MacOS builds
	echo "  \"download_darwin_amd64\": \"$(BASE_URL)/AdGuardHome_darwin_amd64.zip\"," >> $(DIST_DIR)/version.json
	echo "  \"download_darwin_386\": \"$(BASE_URL)/AdGuardHome_darwin_386.zip\"," >> $(DIST_DIR)/version.json

	# Linux
	echo "  \"download_linux_amd64\": \"$(BASE_URL)/AdGuardHome_linux_amd64.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_386\": \"$(BASE_URL)/AdGuardHome_linux_386.tar.gz\"," >> $(DIST_DIR)/version.json

	# Linux, all kinds of ARM
	echo "  \"download_linux_arm\": \"$(BASE_URL)/AdGuardHome_linux_armv6.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_armv5\": \"$(BASE_URL)/AdGuardHome_linux_armv5.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_armv6\": \"$(BASE_URL)/AdGuardHome_linux_armv6.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_armv7\": \"$(BASE_URL)/AdGuardHome_linux_armv7.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_arm64\": \"$(BASE_URL)/AdGuardHome_linux_arm64.tar.gz\"," >> $(DIST_DIR)/version.json

	# Linux, MIPS
	echo "  \"download_linux_mips\": \"$(BASE_URL)/AdGuardHome_linux_mips_softfloat.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_mipsle\": \"$(BASE_URL)/AdGuardHome_linux_mipsle_softfloat.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_mips64\": \"$(BASE_URL)/AdGuardHome_linux_mips64_softfloat.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_linux_mips64le\": \"$(BASE_URL)/AdGuardHome_linux_mips64le_softfloat.tar.gz\"," >> $(DIST_DIR)/version.json

	# FreeBSD
	echo "  \"download_freebsd_386\": \"$(BASE_URL)/AdGuardHome_freebsd_386.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_freebsd_amd64\": \"$(BASE_URL)/AdGuardHome_freebsd_amd64.tar.gz\"," >> $(DIST_DIR)/version.json

	# FreeBSD, all kinds of ARM
	echo "  \"download_freebsd_arm\": \"$(BASE_URL)/AdGuardHome_freebsd_armv6.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_freebsd_armv5\": \"$(BASE_URL)/AdGuardHome_freebsd_armv5.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_freebsd_armv6\": \"$(BASE_URL)/AdGuardHome_freebsd_armv6.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_freebsd_armv7\": \"$(BASE_URL)/AdGuardHome_freebsd_armv7.tar.gz\"," >> $(DIST_DIR)/version.json
	echo "  \"download_freebsd_arm64\": \"$(BASE_URL)/AdGuardHome_freebsd_arm64.tar.gz\"" >> $(DIST_DIR)/version.json

	# Finish
	echo "}" >> $(DIST_DIR)/version.json
endef
