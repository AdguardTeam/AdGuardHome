package main

import (
	"fmt"
	"net"
	"os"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/hmage/golibs/log"
	"github.com/joomcode/errorx"
)

var dnsServer *dnsforward.Server

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer(baseDir string) {
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		log.Fatalf("Cannot create DNS data dir at %s: %s", baseDir, err)
	}

	dnsServer = dnsforward.NewServer(baseDir)
}

func isRunning() bool {
	return dnsServer != nil && dnsServer.IsRunning()
}

func generateServerConfig() dnsforward.ServerConfig {
	filters := []dnsfilter.Filter{}
	userFilter := userFilter()
	filters = append(filters, dnsfilter.Filter{
		ID:    userFilter.ID,
		Rules: userFilter.Rules,
	})
	for _, filter := range config.Filters {
		filters = append(filters, dnsfilter.Filter{
			ID:    filter.ID,
			Rules: filter.Rules,
		})
	}

	newconfig := dnsforward.ServerConfig{
		UDPListenAddr:   &net.UDPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		TCPListenAddr:   &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		FilteringConfig: config.DNS.FilteringConfig,
		Filters:         filters,
	}

	if config.TLS.Enabled {
		newconfig.TLSConfig = config.TLS.TLSConfig
		if config.TLS.PortDNSOverTLS != 0 {
			newconfig.TLSListenAddr = &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.TLS.PortDNSOverTLS}
		}
	}

	for _, u := range config.DNS.UpstreamDNS {
		dnsUpstream, err := upstream.AddressToUpstream(u, config.DNS.BootstrapDNS, dnsforward.DefaultTimeout)
		if err != nil {
			log.Printf("Couldn't get upstream: %s", err)
			// continue, just ignore the upstream
			continue
		}
		newconfig.Upstreams = append(newconfig.Upstreams, dnsUpstream)
	}
	return newconfig
}

func startDNSServer() error {
	if isRunning() {
		return fmt.Errorf("unable to start forwarding DNS server: Already running")
	}

	newconfig := generateServerConfig()
	err := dnsServer.Start(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func reconfigureDNSServer() error {
	if !isRunning() {
		return fmt.Errorf("Refusing to reconfigure forwarding DNS server: not running")
	}

	config := generateServerConfig()
	err := dnsServer.Reconfigure(&config)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func stopDNSServer() error {
	if !isRunning() {
		return fmt.Errorf("Refusing to stop forwarding DNS server: not running")
	}

	err := dnsServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop forwarding DNS server")
	}

	return nil
}
