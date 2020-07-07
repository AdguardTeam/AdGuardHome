NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
GOPATH := $(shell go env GOPATH)
PWD := $(shell pwd)

TARGET=AdGuardHome

# Docker target parameters
DOCKER_IMAGE_NAME ?= adguard/adguardhome
DOCKER_PLATFORMS=linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64,linux/386,linux/ppc64le,linux/s390x
DOCKER_OUTPUT ?= type=image,push=false
DOCKER_TAGS ?= --tag $(DOCKER_IMAGE_NAME):$(VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# See release and snapshot targets
DIST_DIR=dist

# Version and channel (see build and version targets)
TAG_NAME=$(shell git describe --abbrev=0)
# remove leading "v"
RELEASE_VERSION=$(TAG_NAME:v%=%)
COMMIT=$(shell git rev-parse --short HEAD)
SNAPSHOT_VERSION=$(RELEASE_VERSION)-SNAPSHOT-$(COMMIT)
VERSION=$(SNAPSHOT_VERSION)
CHANNEL ?= release
BASE_URL="https://static.adguard.com/adguardhome/$(CHANNEL)"

.PHONY: all build client docker release version clean
all: build

build: client
	go mod download
	go generate ./...
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION) -X main.channel=$(CHANNEL) -X main.goarm=$(GOARM)"
	PATH=$(GOPATH)/bin:$(PATH) packr clean

client:
	npm --prefix client ci
	npm --prefix client run build-prod

docker:
	DOCKER_CLI_EXPERIMENTAL=enabled \
	docker buildx build \
	--platform $(DOCKER_PLATFORMS) \
	--build-arg VERSION=$(VERSION) \
	--build-arg CHANNEL=$(CHANNEL) \
	--build-arg VCS_REF=$(COMMIT) \
	--build-arg BUILD_DATE=$(BUILD_DATE) \
	$(DOCKER_TAGS) \
	--output "$(DOCKER_OUTPUT)" \
	-t "$(DOCKER_IMAGE_NAME)" -f ./Dockerfile .

	@echo Now you can run the docker image:
	@echo docker run --name "$(DOCKER_IMAGE_NAME)" -p 53:53/tcp -p 53:53/udp -p 80:80/tcp -p 443:443/tcp -p 853:853/tcp -p 3000:3000/tcp $(DOCKER_IMAGE_NAME)

snapshot:
	@echo Starting snapshot build: version $(SNAPSHOT_VERSION), channel $(CHANNEL)
	CHANNEL=$(CHANNEL) goreleaser release --rm-dist --skip-publish --snapshot
	$(call write_version_file,$(SNAPSHOT_VERSION))
	PATH=$(GOPATH)/bin:$(PATH) packr clean

release:
	@echo Starting release build: version $(RELEASE_VERSION), channel $(CHANNEL)
	CHANNEL=$(CHANNEL) goreleaser release --rm-dist --skip-publish
	$(call write_version_file,$(RELEASE_VERSION))
	PATH=$(GOPATH)/bin:$(PATH) packr clean

snapshot-docker:
	docker run \
		  	-it \
			--rm \
			-v $(PWD):/build \
		   	-e CHANNEL=$(CHANNEL) \
			golang-ubuntu \
			make snapshot

release-docker:
	docker run \
			-it \
			--rm \
			-v $(PWD):/build \
			-e CHANNEL=$(CHANNEL) \
			golang-ubuntu \
			make release

clean:
	# make build output
	rm -f AdGuardHome
	rm -f AdGuardHome.exe
	# static build output
	rm -rf build
	# dist folder
	rm -rf $(DIST_DIR)
	# client deps
	rm -rf client/node_modules
	# packr-generated files
	PATH=$(GOPATH)/bin:$(PATH) packr clean

define write_version_file
	$(eval version := $(1))

	@echo Writing version file: $(version)

	# Variables for CI
	rm -f $(DIST_DIR)/version.txt
	echo "version=v$(version)" > $(DIST_DIR)/version.txt

	# Prepare the version.json file
	rm -f $(DIST_DIR)/version.json
	echo "{" >> $(DIST_DIR)/version.json
	echo "  \"version\": \"v$(version)\"," >> $(DIST_DIR)/version.json
	echo "  \"announcement\": \"AdGuard Home $(version) is now available!\"," >> $(DIST_DIR)/version.json
	echo "  \"announcement_url\": \"https://github.com/AdguardTeam/AdGuardHome/releases\"," >> $(DIST_DIR)/version.json

	# Windows builds
	echo "  \"download_windows_amd64\": \"$(BASE_URL)/AdGuardHome_windows_amd64.zip\"," >> $(DIST_DIR)/version.json
	echo "  \"download_windows_386\": \"$(BASE_URL)/AdGuardHome_windows_386.zip\"," >> $(DIST_DIR)/version.json

	# MacOS builds
	echo "  \"download_darwin_386\": \"$(BASE_URL)/AdGuardHome_darwin_386.zip\"," >> $(DIST_DIR)/version.json
	echo "  \"download_darwin_amd64\": \"$(BASE_URL)/AdGuardHome_darwin_amd64.zip\"," >> $(DIST_DIR)/version.json

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
	echo "  \"selfupdate_min_version\": \"v0.0\"" >> $(DIST_DIR)/version.json
	echo "}" >> $(DIST_DIR)/version.json
endef