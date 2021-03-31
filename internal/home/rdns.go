package home

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// RDNS resolves clients' addresses to enrich their metadata.
type RDNS struct {
	dnsServer      *dnsforward.Server
	clients        *clientsContainer
	subnetDetector *aghnet.SubnetDetector
	localResolvers aghnet.Exchanger

	// ipCh used to pass client's IP to rDNS workerLoop.
	ipCh chan net.IP

	// ipCache caches the IP addresses to be resolved by rDNS.  The resolved
	// address stays here while it's inside clients.  After leaving clients
	// the address will be resolved once again.  If the address couldn't be
	// resolved, cache prevents further attempts to resolve it for some
	// time.
	ipCache cache.Cache
}

// Default rDNS values.
const (
	defaultRDNSCacheSize = 10000
	defaultRDNSCacheTTL  = 1 * 60 * 60
	defaultRDNSIPChSize  = 256
)

// NewRDNS creates and returns initialized RDNS.
func NewRDNS(
	dnsServer *dnsforward.Server,
	clients *clientsContainer,
	snd *aghnet.SubnetDetector,
	lr aghnet.Exchanger,
) (rDNS *RDNS) {
	rDNS = &RDNS{
		dnsServer:      dnsServer,
		clients:        clients,
		subnetDetector: snd,
		localResolvers: lr,
		ipCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  defaultRDNSCacheSize,
		}),
		ipCh: make(chan net.IP, defaultRDNSIPChSize),
	}

	go rDNS.workerLoop()

	return rDNS
}

// Begin adds the ip to the resolving queue if it is not cached or already
// resolved.
func (r *RDNS) Begin(ip net.IP) {
	now := uint64(time.Now().Unix())
	if expire := r.ipCache.Get(ip); len(expire) != 0 {
		if binary.BigEndian.Uint64(expire) > now {
			return
		}
	}

	// The cache entry either expired or doesn't exist.
	ttl := make([]byte, 8)
	binary.BigEndian.PutUint64(ttl, now+defaultRDNSCacheTTL)
	r.ipCache.Set(ip, ttl)

	id := ip.String()
	if r.clients.Exists(id, ClientSourceRDNS) {
		return
	}

	select {
	case r.ipCh <- ip:
		log.Tracef("rdns: %q added to queue", ip)
	default:
		log.Tracef("rdns: queue is full")
	}
}

const (
	// rDNSEmptyAnswerErr is returned by RDNS resolve method when the answer
	// section of respond is empty.
	rDNSEmptyAnswerErr agherr.Error = "the answer section is empty"

	// rDNSNotPTRErr is returned by RDNS resolve method when the response is
	// not of PTR type.
	rDNSNotPTRErr agherr.Error = "the response is not a ptr"
)

// resolve tries to resolve the ip in a suitable way.
func (r *RDNS) resolve(ip net.IP) (host string, err error) {
	log.Tracef("rdns: resolving host for %q", ip)

	arpa := dns.Fqdn(aghnet.ReverseAddr(ip))
	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Compress: true,
		Question: []dns.Question{{
			Name:   arpa,
			Qtype:  dns.TypePTR,
			Qclass: dns.ClassINET,
		}},
	}

	var resp *dns.Msg
	if r.subnetDetector.IsLocallyServedNetwork(ip) {
		resp, err = r.localResolvers.Exchange(msg)
	} else {
		resp, err = r.dnsServer.Exchange(msg)
	}
	if err != nil {
		return "", fmt.Errorf("performing lookup for %q: %w", arpa, err)
	}

	if len(resp.Answer) == 0 {
		return "", fmt.Errorf("lookup for %q: %w", arpa, rDNSEmptyAnswerErr)
	}

	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		return "", fmt.Errorf("type checking: %w", rDNSNotPTRErr)
	}

	log.Tracef("rdns: ptr response for %q: %s", ip, ptr.String())

	return strings.TrimSuffix(ptr.Ptr, "."), nil
}

// workerLoop handles incoming IP addresses from ipChan and adds it into
// clients.
func (r *RDNS) workerLoop() {
	defer agherr.LogPanic("rdns")

	for ip := range r.ipCh {
		host, err := r.resolve(ip)
		if err != nil {
			log.Error("rdns: resolving %q: %s", ip, err)

			continue
		}

		// Don't handle any errors since AddHost doesn't return non-nil
		// errors for now.
		_, _ = r.clients.AddHost(ip.String(), host, ClientSourceRDNS)
	}
}
