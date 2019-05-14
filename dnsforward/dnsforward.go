package dnsforward

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

// DefaultTimeout is the default upstream timeout
const DefaultTimeout = 10 * time.Second

const (
	safeBrowsingBlockHost = "standard-block.dns.adguard.com"
	parentalBlockHost     = "family-block.dns.adguard.com"
)

// Server is the main way to start a DNS server.
//
// Example:
//  s := dnsforward.Server{}
//  err := s.Start(nil) // will start a DNS server listening on default port 53, in a goroutine
//  err := s.Reconfigure(ServerConfig{UDPListenAddr: &net.UDPAddr{Port: 53535}}) // will reconfigure running DNS server to listen on UDP port 53535
//  err := s.Stop() // will stop listening on port 53535 and cancel all goroutines
//  err := s.Start(nil) // will start listening again, on port 53535, in a goroutine
//
// The zero Server is empty and ready for use.
type Server struct {
	dnsProxy  *proxy.Proxy         // DNS proxy instance
	dnsFilter *dnsfilter.Dnsfilter // DNS filter instance
	queryLog  *queryLog            // Query log instance
	stats     *stats               // General server statistics
	once      sync.Once

	sync.RWMutex
	ServerConfig
}

// NewServer creates a new instance of the dnsforward.Server
// baseDir is the base directory for query logs
func NewServer(baseDir string) *Server {
	return &Server{
		queryLog: newQueryLog(baseDir),
		stats:    newStats(),
	}
}

// FilteringConfig represents the DNS filtering configuration of AdGuard Home
// The zero FilteringConfig is empty and ready for use.
type FilteringConfig struct {
	ProtectionEnabled  bool     `yaml:"protection_enabled"`   // whether or not use any of dnsfilter features
	FilteringEnabled   bool     `yaml:"filtering_enabled"`    // whether or not use filter lists
	BlockingMode       string   `yaml:"blocking_mode"`        // mode how to answer filtered requests
	BlockedResponseTTL uint32   `yaml:"blocked_response_ttl"` // if 0, then default is used (3600)
	QueryLogEnabled    bool     `yaml:"querylog_enabled"`     // if true, query log is enabled
	Ratelimit          int      `yaml:"ratelimit"`            // max number of requests per second from a given IP (0 to disable)
	RatelimitWhitelist []string `yaml:"ratelimit_whitelist"`  // a list of whitelisted client IP addresses
	RefuseAny          bool     `yaml:"refuse_any"`           // if true, refuse ANY requests
	BootstrapDNS       []string `yaml:"bootstrap_dns"`        // a list of bootstrap DNS for DoH and DoT (plain DNS only)
	AllServers         bool     `yaml:"all_servers"`          // if true, parallel queries to all configured upstream servers are enabled

	dnsfilter.Config `yaml:",inline"`
}

// TLSConfig is the TLS configuration for HTTPS, DNS-over-HTTPS, and DNS-over-TLS
type TLSConfig struct {
	TLSListenAddr    *net.TCPAddr `yaml:"-" json:"-"`
	CertificateChain string       `yaml:"certificate_chain" json:"certificate_chain"` // PEM-encoded certificates chain
	PrivateKey       string       `yaml:"private_key" json:"private_key"`             // PEM-encoded private key
}

// ServerConfig represents server configuration.
// The zero ServerConfig is empty and ready for use.
type ServerConfig struct {
	UDPListenAddr            *net.UDPAddr                   // UDP listen address
	TCPListenAddr            *net.TCPAddr                   // TCP listen address
	Upstreams                []upstream.Upstream            // Configured upstreams
	DomainsReservedUpstreams map[string][]upstream.Upstream // Map of domains and lists of configured upstreams
	Filters                  []dnsfilter.Filter             // A list of filters to use

	FilteringConfig
	TLSConfig
}

// if any of ServerConfig values are zero, then default values from below are used
var defaultValues = ServerConfig{
	UDPListenAddr:   &net.UDPAddr{Port: 53},
	TCPListenAddr:   &net.TCPAddr{Port: 53},
	FilteringConfig: FilteringConfig{BlockedResponseTTL: 3600},
}

func init() {
	defaultDNS := []string{"8.8.8.8:53", "8.8.4.4:53"}

	defaultUpstreams := make([]upstream.Upstream, 0)
	for _, addr := range defaultDNS {
		u, err := upstream.AddressToUpstream(addr, upstream.Options{Timeout: DefaultTimeout})
		if err == nil {
			defaultUpstreams = append(defaultUpstreams, u)
		}
	}
	defaultValues.Upstreams = defaultUpstreams
}

// Start starts the DNS server
func (s *Server) Start(config *ServerConfig) error {
	s.Lock()
	defer s.Unlock()
	return s.startInternal(config)
}

// startInternal starts without locking
func (s *Server) startInternal(config *ServerConfig) error {
	if config != nil {
		s.ServerConfig = *config
	}

	if s.dnsFilter != nil || s.dnsProxy != nil {
		return errors.New("DNS server is already started")
	}

	if s.queryLog == nil {
		s.queryLog = newQueryLog(".")
	}

	if s.stats == nil {
		s.stats = newStats()
	}

	err := s.initDNSFilter()
	if err != nil {
		return err
	}

	log.Tracef("Loading stats from querylog")
	err = s.queryLog.fillStatsFromQueryLog(s.stats)
	if err != nil {
		return errorx.Decorate(err, "failed to load stats from querylog")
	}

	// TODO: Think about reworking this, the current approach won't work properly if AG Home is restarted periodically
	s.once.Do(func() {
		log.Printf("Start DNS server periodic jobs")
		go s.queryLog.periodicQueryLogRotate()
		go s.queryLog.runningTop.periodicHourlyTopRotate()
		go s.stats.statsRotator()
	})

	proxyConfig := proxy.Config{
		UDPListenAddr:            s.UDPListenAddr,
		TCPListenAddr:            s.TCPListenAddr,
		Ratelimit:                s.Ratelimit,
		RatelimitWhitelist:       s.RatelimitWhitelist,
		RefuseAny:                s.RefuseAny,
		CacheEnabled:             true,
		Upstreams:                s.Upstreams,
		DomainsReservedUpstreams: s.DomainsReservedUpstreams,
		Handler:                  s.handleDNSRequest,
		AllServers:               s.AllServers,
	}

	if s.TLSListenAddr != nil && s.CertificateChain != "" && s.PrivateKey != "" {
		proxyConfig.TLSListenAddr = s.TLSListenAddr
		keypair, err := tls.X509KeyPair([]byte(s.CertificateChain), []byte(s.PrivateKey))
		if err != nil {
			return errorx.Decorate(err, "Failed to parse TLS keypair")
		}
		proxyConfig.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}
	}

	if proxyConfig.UDPListenAddr == nil {
		proxyConfig.UDPListenAddr = defaultValues.UDPListenAddr
	}

	if proxyConfig.TCPListenAddr == nil {
		proxyConfig.TCPListenAddr = defaultValues.TCPListenAddr
	}

	if len(proxyConfig.Upstreams) == 0 {
		proxyConfig.Upstreams = defaultValues.Upstreams
	}

	// Initialize and start the DNS proxy
	s.dnsProxy = &proxy.Proxy{Config: proxyConfig}
	return s.dnsProxy.Start()
}

// Initializes the DNS filter
func (s *Server) initDNSFilter() error {
	log.Tracef("Creating dnsfilter")
	s.dnsFilter = dnsfilter.New(&s.Config)
	// add rules only if they are enabled
	if s.FilteringEnabled {
		err := s.dnsFilter.AddRules(s.Filters)
		if err != nil {
			return errorx.Decorate(err, "could not initialize dnsfilter")
		}
	}
	return nil
}

// Stop stops the DNS server
func (s *Server) Stop() error {
	s.Lock()
	defer s.Unlock()
	return s.stopInternal()
}

// stopInternal stops without locking
func (s *Server) stopInternal() error {
	if s.dnsProxy != nil {
		err := s.dnsProxy.Stop()
		s.dnsProxy = nil
		if err != nil {
			return errorx.Decorate(err, "could not stop the DNS server properly")
		}
	}

	if s.dnsFilter != nil {
		s.dnsFilter.Destroy()
		s.dnsFilter = nil
	}

	// flush remainder to file
	return s.queryLog.flushLogBuffer()
}

// IsRunning returns true if the DNS server is running
func (s *Server) IsRunning() bool {
	s.RLock()
	isRunning := true
	if s.dnsProxy == nil {
		isRunning = false
	}
	s.RUnlock()
	return isRunning
}

// Reconfigure applies the new configuration to the DNS server
func (s *Server) Reconfigure(config *ServerConfig) error {
	s.Lock()
	defer s.Unlock()

	log.Print("Start reconfiguring the server")
	err := s.stopInternal()
	if err != nil {
		return errorx.Decorate(err, "could not reconfigure the server")
	}
	err = s.startInternal(config)
	if err != nil {
		return errorx.Decorate(err, "could not reconfigure the server")
	}

	return nil
}

// ServeHTTP is a HTTP handler method we use to provide DNS-over-HTTPS
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	s.dnsProxy.ServeHTTP(w, r)
	s.RUnlock()
}

// GetQueryLog returns a map with the current query log ready to be converted to a JSON
func (s *Server) GetQueryLog() []map[string]interface{} {
	s.RLock()
	defer s.RUnlock()
	return s.queryLog.getQueryLog()
}

// GetStatsTop returns the current stop stats
func (s *Server) GetStatsTop() *StatsTop {
	s.RLock()
	defer s.RUnlock()
	return s.queryLog.runningTop.getStatsTop()
}

// PurgeStats purges current server stats
func (s *Server) PurgeStats() {
	s.Lock()
	defer s.Unlock()
	s.stats.purgeStats()
}

// GetAggregatedStats returns aggregated stats data for the 24 hours
func (s *Server) GetAggregatedStats() map[string]interface{} {
	s.RLock()
	defer s.RUnlock()
	return s.stats.getAggregatedStats()
}

// GetStatsHistory gets stats history aggregated by the specified time unit
// timeUnit is either time.Second, time.Minute, time.Hour, or 24*time.Hour
// start is start of the time range
// end is end of the time range
// returns nil if time unit is not supported
func (s *Server) GetStatsHistory(timeUnit time.Duration, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	s.RLock()
	defer s.RUnlock()
	return s.stats.getStatsHistory(timeUnit, startTime, endTime)
}

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
func (s *Server) handleDNSRequest(p *proxy.Proxy, d *proxy.DNSContext) error {
	start := time.Now()

	// use dnsfilter before cache -- changed settings or filters would require cache invalidation otherwise
	res, err := s.filterDNSRequest(d)
	if err != nil {
		return err
	}

	if d.Res == nil {
		// request was not filtered so let it be processed further
		err = p.Resolve(d)
		if err != nil {
			return err
		}
	}

	shouldLog := true
	msg := d.Req

	// don't log ANY request if refuseAny is enabled
	if len(msg.Question) >= 1 && msg.Question[0].Qtype == dns.TypeANY && s.RefuseAny {
		shouldLog = false
	}

	if s.QueryLogEnabled && shouldLog {
		elapsed := time.Since(start)
		upstreamAddr := ""
		if d.Upstream != nil {
			upstreamAddr = d.Upstream.Address()
		}
		entry := s.queryLog.logRequest(msg, d.Res, res, elapsed, d.Addr, upstreamAddr)
		if entry != nil {
			s.stats.incrementCounters(entry)
		}
	}

	return nil
}

// filterDNSRequest applies the dnsFilter and sets d.Res if the request was filtered
func (s *Server) filterDNSRequest(d *proxy.DNSContext) (*dnsfilter.Result, error) {
	msg := d.Req
	host := strings.TrimSuffix(msg.Question[0].Name, ".")

	s.RLock()
	protectionEnabled := s.ProtectionEnabled
	dnsFilter := s.dnsFilter
	s.RUnlock()

	if !protectionEnabled {
		return nil, nil
	}

	var res dnsfilter.Result
	var err error

	res, err = dnsFilter.CheckHost(host)
	if err != nil {
		// Return immediately if there's an error
		return nil, errorx.Decorate(err, "dnsfilter failed to check host '%s'", host)
	} else if res.IsFiltered {
		// log.Tracef("Host %s is filtered, reason - '%s', matched rule: '%s'", host, res.Reason, res.Rule)
		d.Res = s.genDNSFilterMessage(d, &res)
	}

	return &res, err
}

// genDNSFilterMessage generates a DNS message corresponding to the filtering result
func (s *Server) genDNSFilterMessage(d *proxy.DNSContext, result *dnsfilter.Result) *dns.Msg {
	m := d.Req

	if m.Question[0].Qtype != dns.TypeA {
		return s.genNXDomain(m)
	}

	switch result.Reason {
	case dnsfilter.FilteredSafeBrowsing:
		return s.genBlockedHost(m, safeBrowsingBlockHost, d)
	case dnsfilter.FilteredParental:
		return s.genBlockedHost(m, parentalBlockHost, d)
	default:
		if result.IP != nil {
			return s.genARecord(m, result.IP)
		}

		if s.BlockingMode == "null_ip" {
			return s.genARecord(m, net.IPv4zero)
		}

		return s.genNXDomain(m)
	}
}

func (s *Server) genServerFailure(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeServerFailure)
	resp.RecursionAvailable = true
	return &resp
}

func (s *Server) genARecord(request *dns.Msg, ip net.IP) *dns.Msg {
	resp := dns.Msg{}
	resp.SetReply(request)
	answer, err := dns.NewRR(fmt.Sprintf("%s %d A %s", request.Question[0].Name, s.BlockedResponseTTL, ip.String()))
	if err != nil {
		log.Printf("Couldn't generate A record for replacement host '%s': %s", ip.String(), err)
		return s.genServerFailure(request)
	}
	resp.Answer = append(resp.Answer, answer)
	return &resp
}

func (s *Server) genBlockedHost(request *dns.Msg, newAddr string, d *proxy.DNSContext) *dns.Msg {
	// look up the hostname, TODO: cache
	replReq := dns.Msg{}
	replReq.SetQuestion(dns.Fqdn(newAddr), request.Question[0].Qtype)
	replReq.RecursionDesired = true

	newContext := &proxy.DNSContext{
		Proto:     d.Proto,
		Addr:      d.Addr,
		StartTime: time.Now(),
		Req:       &replReq,
	}

	err := s.dnsProxy.Resolve(newContext)
	if err != nil {
		log.Printf("Couldn't look up replacement host '%s': %s", newAddr, err)
		return s.genServerFailure(request)
	}

	resp := dns.Msg{}
	resp.SetReply(request)
	resp.Authoritative, resp.RecursionAvailable = true, true
	if newContext.Res != nil {
		for _, answer := range newContext.Res.Answer {
			answer.Header().Name = request.Question[0].Name
			resp.Answer = append(resp.Answer, answer)
		}
	}

	return &resp
}

func (s *Server) genNXDomain(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeNameError)
	resp.RecursionAvailable = true
	resp.Ns = s.genSOA(request)
	return &resp
}

func (s *Server) genSOA(request *dns.Msg) []dns.RR {
	zone := ""
	if len(request.Question) > 0 {
		zone = request.Question[0].Name
	}

	soa := dns.SOA{
		// values copied from verisign's nonexistent .com domain
		// their exact values are not important in our use case because they are used for domain transfers between primary/secondary DNS servers
		Refresh: 1800,
		Retry:   900,
		Expire:  604800,
		Minttl:  86400,
		// copied from AdGuard DNS
		Ns:     "fake-for-negative-caching.adguard.com.",
		Serial: 100500,
		// rest is request-specific
		Hdr: dns.RR_Header{
			Name:   zone,
			Rrtype: dns.TypeSOA,
			Ttl:    s.BlockedResponseTTL,
			Class:  dns.ClassINET,
		},
		Mbox: "hostmaster.", // zone will be appended later if it's not empty or "."
	}
	if soa.Hdr.Ttl == 0 {
		soa.Hdr.Ttl = defaultValues.BlockedResponseTTL
	}
	if len(zone) > 0 && zone[0] != '.' {
		soa.Mbox += zone
	}
	return []dns.RR{&soa}
}
