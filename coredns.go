package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync" // Include all plugins.

	_ "github.com/AdguardTeam/AdGuardHome/coredns_plugin"
	_ "github.com/AdguardTeam/AdGuardHome/coredns_plugin/ratelimit"
	_ "github.com/AdguardTeam/AdGuardHome/upstream"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
	_ "github.com/coredns/coredns/plugin/auto"
	_ "github.com/coredns/coredns/plugin/autopath"
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/chaos"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/dnssec"
	_ "github.com/coredns/coredns/plugin/dnstap"
	_ "github.com/coredns/coredns/plugin/erratic"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/file"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/health"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/loadbalance"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/loop"
	_ "github.com/coredns/coredns/plugin/metadata"
	_ "github.com/coredns/coredns/plugin/metrics"
	_ "github.com/coredns/coredns/plugin/nsid"
	_ "github.com/coredns/coredns/plugin/pprof"
	_ "github.com/coredns/coredns/plugin/proxy"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/coredns/plugin/rewrite"
	_ "github.com/coredns/coredns/plugin/root"
	_ "github.com/coredns/coredns/plugin/secondary"
	_ "github.com/coredns/coredns/plugin/template"
	_ "github.com/coredns/coredns/plugin/tls"
	_ "github.com/coredns/coredns/plugin/whoami"
	_ "github.com/mholt/caddy/onevent"
)

// Directives are registered in the order they should be
// executed.
//
// Ordering is VERY important. Every plugin will
// feel the effects of all other plugin below
// (after) them during a request, but they must not
// care what plugin above them are doing.

var directives = []string{
	"metadata",
	"tls",
	"reload",
	"nsid",
	"root",
	"bind",
	"debug",
	"health",
	"pprof",
	"prometheus",
	"errors",
	"log",
	"ratelimit",
	"dnsfilter",
	"dnstap",
	"chaos",
	"loadbalance",
	"cache",
	"rewrite",
	"dnssec",
	"autopath",
	"template",
	"hosts",
	"file",
	"auto",
	"secondary",
	"loop",
	"forward",
	"proxy",
	"upstream",
	"erratic",
	"whoami",
	"on",
}

func init() {
	dnsserver.Directives = directives
}

var (
	isCoreDNSRunningLock sync.Mutex
	isCoreDNSRunning     = false
)

func isRunning() bool {
	isCoreDNSRunningLock.Lock()
	value := isCoreDNSRunning
	isCoreDNSRunningLock.Unlock()
	return value
}

func startDNSServer() error {
	isCoreDNSRunningLock.Lock()
	if isCoreDNSRunning {
		isCoreDNSRunningLock.Unlock()
		return fmt.Errorf("Unable to start coreDNS: Already running")
	}
	isCoreDNSRunning = true
	isCoreDNSRunningLock.Unlock()

	configpath := filepath.Join(config.ourBinaryDir, config.CoreDNS.coreFile)
	os.Args = os.Args[:1]
	os.Args = append(os.Args, "-conf")
	os.Args = append(os.Args, configpath)

	err := writeCoreDNSConfig()
	if err != nil {
		errortext := fmt.Errorf("Unable to write coredns config: %s", err)
		log.Println(errortext)
		return errortext
	}

	go coremain.Run()
	return nil
}
