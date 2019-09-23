package home

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const (
	rdnsTimeout = 3 * time.Second // max time to wait for rDNS response
)

// RDNS - module context
type RDNS struct {
	clients   *clientsContainer
	ipChannel chan string // pass data from DNS request handling thread to rDNS thread
	// contains IP addresses of clients to be resolved by rDNS
	// if IP address couldn't be resolved, it stays here forever to prevent further attempts to resolve the same IP
	ips      map[string]bool
	lock     sync.Mutex        // synchronize access to 'ips'
	upstream upstream.Upstream // Upstream object for our own DNS server
}

// InitRDNS - create module context
func InitRDNS(clients *clientsContainer) *RDNS {
	r := RDNS{}
	r.clients = clients
	var err error

	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	resolverAddress := fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)

	opts := upstream.Options{
		Timeout: rdnsTimeout,
	}
	r.upstream, err = upstream.AddressToUpstream(resolverAddress, opts)
	if err != nil {
		log.Error("upstream.AddressToUpstream: %s", err)
		return nil
	}

	r.ips = make(map[string]bool)
	r.ipChannel = make(chan string, 256)
	go r.workerLoop()
	return &r
}

// Begin - add IP address to rDNS queue
func (r *RDNS) Begin(ip string) {
	if r.clients.Exists(ip, ClientSourceRDNS) {
		return
	}

	// add IP to ips, if not exists
	r.lock.Lock()
	defer r.lock.Unlock()
	_, ok := r.ips[ip]
	if ok {
		return
	}
	r.ips[ip] = true

	log.Tracef("Adding %s for rDNS resolve", ip)
	select {
	case r.ipChannel <- ip:
		//
	default:
		log.Tracef("rDNS queue is full")
	}
}

// Use rDNS to get hostname by IP address
func (r *RDNS) resolve(ip string) string {
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

	resp, err := r.upstream.Exchange(&req)
	if err != nil {
		log.Debug("Error while making an rDNS lookup for %s: %s", ip, err)
		return ""
	}
	if len(resp.Answer) != 1 {
		log.Debug("No answer for rDNS lookup of %s", ip)
		return ""
	}
	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		log.Debug("not a PTR response for %s", ip)
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
func (r *RDNS) workerLoop() {
	for {
		var ip string
		ip = <-r.ipChannel

		host := r.resolve(ip)
		if len(host) == 0 {
			continue
		}

		r.lock.Lock()
		delete(r.ips, ip)
		r.lock.Unlock()

		_, _ = config.clients.AddHost(ip, host, ClientSourceRDNS)
	}
}
