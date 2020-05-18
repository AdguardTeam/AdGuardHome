GIT_VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
GOPATH := $(shell go env GOPATH)
JSFILES = $(shell find client -path client/node_modules -prune -o -type f -name '*.js')
STATIC = build/static/index.html
CHANNEL ?= release
DOCKER_IMAGE_DEV_NAME=adguardhome-dev
DOCKERFILE=packaging/docker/Dockerfile
DOCKERFILE_HUB=packaging/docker/Dockerfile.travis

TARGET=AdGuardHome

.PHONY: all build clean
all: build

build: $(TARGET)

client/node_modules: client/package.json client/package-lock.json
	npm --prefix client ci
	touch client/node_modules

$(STATIC): $(JSFILES) client/node_modules
	npm --prefix client run build-prod

$(TARGET): $(STATIC) *.go home/*.go dhcpd/*.go dnsfilter/*.go dnsforward/*.go
	GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) GO111MODULE=off go get -v github.com/gobuffalo/packr/...
	PATH=$(GOPATH)/bin:$(PATH) packr -z
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(GIT_VERSION) -X main.channel=$(CHANNEL) -X main.goarm=$(GOARM)" -asmflags="-trimpath=$(PWD)" -gcflags="-trimpath=$(PWD)"
	PATH=$(GOPATH)/bin:$(PATH) packr clean

docker:
	docker build -t "$(DOCKER_IMAGE_DEV_NAME)" -f "$(DOCKERFILE)" .
	@echo Now you can run the docker image:
	@echo docker run --name "$(DOCKER_IMAGE_DEV_NAME)" -p 53:53/tcp -p 53:53/udp -p 80:80/tcp -p 443:443/tcp -p 853:853/tcp -p 3000:3000/tcp $(DOCKER_IMAGE_DEV_NAME)

clean:
	$(MAKE) cleanfast
	rm -rf build
	rm -rf client/node_modules

cleanfast:
	rm -f $(TARGET)
