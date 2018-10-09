GIT_VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(patsubst %/,%,$(dir $(mkfile_path)))
GOPATH := $(mkfile_dir)/build/gopath
STATIC := build/static/index.html

.PHONY: all build clean
all: build

build: AdguardDNS coredns

client/node_modules: client/package.json client/package-lock.json
	npm --prefix client install
	touch client/node_modules

$(STATIC): client/node_modules
	npm --prefix client run build-prod

AdguardDNS: $(STATIC) *.go
	mkdir -p $(GOPATH)/src/github.com/AdguardTeam
	if [ ! -h $(GOPATH)/src/github.com/AdguardTeam/AdguardDNS ]; then rm -rf $(GOPATH)/src/github.com/AdguardTeam/AdguardDNS && ln -fs ../../../../.. $(GOPATH)/src/github.com/AdguardTeam/AdguardDNS; fi
	GOPATH=$(GOPATH) go get -v -d .
	GOPATH=$(GOPATH) GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) go get -v github.com/gobuffalo/packr/...
	mkdir -p $(GOPATH)/src/github.com/AdguardTeam/AdguardDNS/build/static ## work around packr bug
	GOPATH=$(GOPATH) PATH=$(GOPATH)/bin:$(PATH) packr build -ldflags="-X main.VersionString=$(GIT_VERSION)" -o AdguardDNS

coredns: coredns_plugin/*.go dnsfilter/*.go
	GOPATH=$(GOPATH) go get -v -d github.com/coredns/coredns
	cd $(GOPATH)/src/github.com/prometheus/client_golang && git checkout -q v0.8.0
	cd $(GOPATH)/src/github.com/coredns/coredns && perl -p -i.bak -e 's/^(trace|route53|federation|kubernetes|etcd):.*//' plugin.cfg
	cd $(GOPATH)/src/github.com/coredns/coredns && grep -q '^dnsfilter:' plugin.cfg || perl -p -i.bak -e 's|^log:log|log:log\ndnsfilter:github.com/AdguardTeam/AdguardDNS/coredns_plugin|' plugin.cfg
	grep '^dnsfilter:' $(GOPATH)/src/github.com/coredns/coredns/plugin.cfg ## used to check that plugin.cfg was successfully edited by sed
	perl -0777 -p -i.bak -e 's/pprofOnce.Do\(func\(\) {(.*)}\)/\1/ms' $(GOPATH)/src/github.com/coredns/coredns/plugin/pprof/setup.go
	perl -0777 -p -i.bak -e 's/c.OnShutdown/c.OnRestart/' $(GOPATH)/src/github.com/coredns/coredns/plugin/pprof/setup.go
	cd $(GOPATH)/src/github.com/coredns/coredns && GOPATH=$(GOPATH) GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) go generate
	cd $(GOPATH)/src/github.com/coredns/coredns && GOPATH=$(GOPATH) go get -v -d .
	cd $(GOPATH)/src/github.com/coredns/coredns && GOPATH=$(GOPATH) go build -o $(mkfile_dir)/coredns

clean:
	$(MAKE) cleanfast
	rm -rf build
	rm -rf client/node_modules

cleanfast:
	rm -f coredns AdguardDNS
