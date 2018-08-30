GIT_VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
GOPATH := $(shell go env GOPATH)
NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(patsubst %/,%,$(dir $(mkfile_path)))
STATIC := build/static/bundle.css build/static/bundle.js build/static/index.html

.PHONY: all build clean
all: build

build: AdguardDNS coredns

$(STATIC):
	yarn --cwd client install
	yarn --cwd client run build-prod

AdguardDNS: $(STATIC) *.go
	echo mkfile_dir = $(mkfile_dir)
	go get -v -d .
	GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) go get -v github.com/gobuffalo/packr/...
	PATH=$(GOPATH)/bin:$(PATH) packr build -ldflags="-X main.VersionString=$(GIT_VERSION)" -o AdguardDNS

coredns: coredns_plugin/*.go dnsfilter/*.go
	echo mkfile_dir = $(mkfile_dir)
	go get -v -d github.com/coredns/coredns
	cd $(GOPATH)/src/github.com/coredns/coredns && grep -q 'dnsfilter:' plugin.cfg || sed -E -i.bak $$'s|^log:log|log:log\\\ndnsfilter:github.com/AdguardTeam/AdguardDNS/coredns_plugin|g' plugin.cfg
	cd $(GOPATH)/src/github.com/coredns/coredns && GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) go generate
	cd $(GOPATH)/src/github.com/coredns/coredns && go get -v -d .
	cd $(GOPATH)/src/github.com/coredns/coredns && go build -o $(mkfile_dir)/coredns

clean:
	rm -vf coredns AdguardDNS
