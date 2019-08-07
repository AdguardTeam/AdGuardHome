package home

import (
	"fmt"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const (
	rdnsTimeout = 3 * time.Second // max time to wait for rDNS response
)

func initRDNS() {
	var err error

	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	resolverAddress := fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)

	opts := upstream.Options{
		Timeout: rdnsTimeout,
	}
	config.dnsctx.upstream, err = upstream.AddressToUpstream(resolverAddress, opts)
	if err != nil {
		log.Error("upstream.AddressToUpstream: %s", err)
		return
	}

	config.dnsctx.rdnsIP = make(map[string]bool)
	config.dnsctx.rdnsChannel = make(chan string, 256)
	go asyncRDNSLoop()
}

// Add IP address to the rDNS queue
func beginAsyncRDNS(ip string) {
	if config.clients.Exists(ip) {
		return
	}

	// add IP to rdnsIP, if not exists
	config.dnsctx.rdnsLock.Lock()
	defer config.dnsctx.rdnsLock.Unlock()
	_, ok := config.dnsctx.rdnsIP[ip]
	if ok {
		return
	}
	config.dnsctx.rdnsIP[ip] = true

	log.Tracef("Adding %s for rDNS resolve", ip)
	select {
	case config.dnsctx.rdnsChannel <- ip:
		//
	default:
		log.Tracef("rDNS queue is full")
	}
}

// Use rDNS to get hostname by IP address
func resolveRDNS(ip string) string {
	log.Tracef("Resolving host for %s", ip)

	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{
			Qtype:  dns.TypePTR,
			Qclass: dns.ClassINET,
		},
	}
	var err error
	req.Question[0].Name, err = dns.ReverseAddr(ip)
	if err != nil {
		log.Debug("Error while calling dns.ReverseAddr(%s): %s", ip, err)
		return ""
	}

	resp, err := config.dnsctx.upstream.Exchange(&req)
	if err != nil {
		log.Error("Error while making an rDNS lookup for %s: %s", ip, err)
		return ""
	}
	if len(resp.Answer) != 1 {
		log.Debug("No answer for rDNS lookup of %s", ip)
		return ""
	}
	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		log.Error("not a PTR response for %s", ip)
		return ""
	}

	log.Tracef("PTR response for %s: %s", ip, ptr.String())
	if strings.HasSuffix(ptr.Ptr, ".") {
		ptr.Ptr = ptr.Ptr[:len(ptr.Ptr)-1]
	}

	return ptr.Ptr
}

// Wait for a signal and then synchronously resolve hostname by IP address
// Add the hostname:IP pair to "Clients" array
func asyncRDNSLoop() {
	for {
		var ip string
		ip = <-config.dnsctx.rdnsChannel

		host := resolveRDNS(ip)
		if len(host) == 0 {
			continue
		}

		config.dnsctx.rdnsLock.Lock()
		delete(config.dnsctx.rdnsIP, ip)
		config.dnsctx.rdnsLock.Unlock()

		_, _ = config.clients.AddHost(ip, host, ClientSourceRDNS)
	}
}
