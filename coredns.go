package main

import (
	"fmt"
	"log"
	"net"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/joomcode/errorx"
)

var dnsServer = dnsforward.Server{}

func isRunning() bool {
	return dnsServer.IsRunning()
}

func generateServerConfig() dnsforward.ServerConfig {
	filters := []dnsforward.Filter{}
	userFilter := userFilter()
	filters = append(filters, dnsforward.Filter{
		ID:    userFilter.ID,
		Rules: userFilter.Rules,
	})
	for _, filter := range config.Filters {
		filters = append(filters, dnsforward.Filter{
			ID:    filter.ID,
			Rules: filter.Rules,
		})
	}

	newconfig := dnsforward.ServerConfig{
		UDPListenAddr:      &net.UDPAddr{Port: config.CoreDNS.Port},
		BlockedResponseTTL: config.CoreDNS.BlockedResponseTTL,
		Filters:            filters,
	}

	for _, u := range config.CoreDNS.UpstreamDNS {
		upstream, err := dnsforward.GetUpstream(u)
		if err != nil {
			log.Printf("Couldn't get upstream: %s", err)
			// continue, just ignore the upstream
			continue
		}
		newconfig.Upstreams = append(newconfig.Upstreams, upstream)
	}
	return newconfig
}

func startDNSServer() error {
	if isRunning() {
		return fmt.Errorf("Unable to start forwarding DNS server: Already running")
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

	err := dnsServer.Reconfigure(generateServerConfig())
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}
