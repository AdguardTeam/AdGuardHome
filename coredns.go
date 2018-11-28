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

func startDNSServer() error {
	if isRunning() {
		return fmt.Errorf("Unable to start coreDNS: Already running")
	}

	filters := []dnsforward.Filter{}
	for _, filter := range config.Filters {
		filters = append(filters, dnsforward.Filter{
			ID:    filter.ID,
			Rules: filter.Rules,
		})
	}

	newconfig := dnsforward.ServerConfig{
		UDPListenAddr: &net.UDPAddr{Port: config.CoreDNS.Port},
		BlockedTTL:    config.CoreDNS.BlockedResponseTTL,
		Filters:       filters,
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

	err := dnsServer.Start(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}
