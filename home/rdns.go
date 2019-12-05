package home

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const (
	rdnsTimeout = 3 * time.Second // max time to wait for rDNS response
)

// RDNS - module context
type RDNS struct {
	clients   *clientsContainer
	ipChannel chan string       // pass data from DNS request handling thread to rDNS thread
	upstream  upstream.Upstream // Upstream object for our own DNS server

	// Contains IP addresses of clients to be resolved by rDNS
	// If IP address is resolved, it stays here while it's inside Clients.
	//  If it's removed from Clients, this IP address will be resolved once again.
	// If IP address couldn't be resolved, it stays here for some time to prevent further attempts to resolve the same IP.
	ipAddrs cache.Cache
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

	cconf := cache.Config{}
	cconf.EnableLRU = true
	cconf.MaxCount = 10000
	r.ipAddrs = cache.New(cconf)

	r.ipChannel = make(chan string, 256)
	go r.workerLoop()
	return &r
}

// Begin - add IP address to rDNS queue
func (r *RDNS) Begin(ip string) {
	now := uint64(time.Now().Unix())
	expire := r.ipAddrs.Get([]byte(ip))
	if len(expire) != 0 {
		exp := binary.BigEndian.Uint64(expire)
		if exp > now {
			return
		}
		// TTL expired
	}
	expire = make([]byte, 8)
	const ttl = 1 * 60 * 60
	binary.BigEndian.PutUint64(expire, now+ttl)
	_ = r.ipAddrs.Set([]byte(ip), expire)

	if r.clients.Exists(ip, ClientSourceRDNS) {
		return
	}

	log.Tracef("rDNS: adding %s", ip)
	select {
	case r.ipChannel <- ip:
		//
	default:
		log.Tracef("rDNS: queue is full")
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
	if len(resp.Answer) == 0 {
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

		_, _ = config.clients.AddHost(ip, host, ClientSourceRDNS)
	}
}
