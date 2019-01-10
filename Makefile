GIT_VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
GOPATH := $(shell go env GOPATH)
JSFILES = $(shell find client -path client/node_modules -prune -o -type f -name '*.js')
STATIC = build/static/index.html

TARGET=AdGuardHome

.PHONY: all build clean
all: build

build: $(TARGET)

client/node_modules: client/package.json client/package-lock.json
	npm --prefix client install
	touch client/node_modules

$(STATIC): $(JSFILES) client/node_modules
	npm --prefix client run build-prod

$(TARGET): $(STATIC) *.go dhcpd/*.go dnsfilter/*.go dnsforward/*.go
	go env
	go get -d .
	GOOS=$(HOST_PLATFORM) GOARCH=$(NATIVE_GOARCH) GO111MODULE=off go get -v github.com/gobuffalo/packr/...
	PATH=$(GOPATH)/bin:$(PATH) packr -z
	GOOS=$(HOST_PLATFORM) GOARCH=$(NATIVE_GOARCH) GOARM=6 CGO_ENABLED=0 go build -ldflags="-s -w -X main.VersionString=$(GIT_VERSION)" -asmflags="-trimpath=$(PWD)" -gcflags="-trimpath=$(PWD)"
	PATH=$(GOPATH)/bin:$(PATH) packr clean

clean:
	$(MAKE) cleanfast
	rm -rf build
	rm -rf client/node_modules

cleanfast:
	rm -f $(TARGET)
