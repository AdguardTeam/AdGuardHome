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

	AllowedClients         map[string]bool // IP addresses of whitelist clients
	DisallowedClients      map[string]bool // IP addresses of clients that should be blocked
	AllowedClientsIPNet    []net.IPNet     // CIDRs of whitelist clients
	DisallowedClientsIPNet []net.IPNet     // CIDRs of clients that should be blocked
	BlockedHosts           map[string]bool // hosts that should be blocked

	sync.RWMutex
	conf ServerConfig
}

// NewServer creates a new instance of the dnsforward.Server
// baseDir is the base directory for query logs
// Note: this function must be called only once
func NewServer(baseDir string) *Server {
	s := &Server{
		queryLog: newQueryLog(baseDir),
		stats:    newStats(),
	}

	log.Tracef("Loading stats from querylog")
	err := s.queryLog.fillStatsFromQueryLog(s.stats)
	if err != nil {
		log.Error("failed to load stats from querylog: %s", err)
	}

	log.Printf("Start DNS server periodic jobs")
	go s.queryLog.periodicQueryLogRotate()
	go s.queryLog.runningTop.periodicHourlyTopRotate()
	go s.stats.statsRotator()
	return s
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

	AllowedClients    []string `yaml:"allowed_clients"`    // IP addresses of whitelist clients
	DisallowedClients []string `yaml:"disallowed_clients"` // IP addresses of clients that should be blocked
	BlockedHosts      []string `yaml:"blocked_hosts"`      // hosts that should be blocked

	// IP (or domain name) which is used to respond to DNS requests blocked by parental control or safe-browsing
	ParentalBlockHost     string `yaml:"parental_block_host"`
	SafeBrowsingBlockHost string `yaml:"safebrowsing_block_host"`

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
	OnDNSRequest             func(d *proxy.DNSContext)

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

func convertArrayToMap(dst *map[string]bool, src []string) {
	*dst = make(map[string]bool)
	for _, s := range src {
		(*dst)[s] = true
	}
}

// Split array of IP or CIDR into 2 containers for fast search
func processIPCIDRArray(dst *map[string]bool, dstIPNet *[]net.IPNet, src []string) error {
	*dst = make(map[string]bool)

	for _, s := range src {
		ip := net.ParseIP(s)
		if ip != nil {
			(*dst)[s] = true
			continue
		}

		_, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			return err
		}
		*dstIPNet = append(*dstIPNet, *ipnet)
	}

	return nil
}

// startInternal starts without locking
func (s *Server) startInternal(config *ServerConfig) error {
	if s.dnsFilter != nil || s.dnsProxy != nil {
		return errors.New("DNS server is already started")
	}

	err := s.initDNSFilter(config)
	if err != nil {
		return err
	}

	proxyConfig := proxy.Config{
		UDPListenAddr:            s.conf.UDPListenAddr,
		TCPListenAddr:            s.conf.TCPListenAddr,
		Ratelimit:                s.conf.Ratelimit,
		RatelimitWhitelist:       s.conf.RatelimitWhitelist,
		RefuseAny:                s.conf.RefuseAny,
		CacheEnabled:             true,
		Upstreams:                s.conf.Upstreams,
		DomainsReservedUpstreams: s.conf.DomainsReservedUpstreams,
		BeforeRequestHandler:     s.beforeRequestHandler,
		RequestHandler:           s.handleDNSRequest,
		AllServers:               s.conf.AllServers,
	}

	err = processIPCIDRArray(&s.AllowedClients, &s.AllowedClientsIPNet, s.conf.AllowedClients)
	if err != nil {
		return err
	}

	err = processIPCIDRArray(&s.DisallowedClients, &s.DisallowedClientsIPNet, s.conf.DisallowedClients)
	if err != nil {
		return err
	}

	convertArrayToMap(&s.BlockedHosts, s.conf.BlockedHosts)

	if s.conf.TLSListenAddr != nil && s.conf.CertificateChain != "" && s.conf.PrivateKey != "" {
		proxyConfig.TLSListenAddr = s.conf.TLSListenAddr
		keypair, err := tls.X509KeyPair([]byte(s.conf.CertificateChain), []byte(s.conf.PrivateKey))
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
func (s *Server) initDNSFilter(config *ServerConfig) error {
	log.Tracef("Creating dnsfilter")

	if config != nil {
		s.conf = *config
	}

	var filters map[int]string
	filters = nil
	if s.conf.FilteringEnabled {
		filters = make(map[int]string)
		for _, f := range s.conf.Filters {
			if f.ID == 0 {
				filters[int(f.ID)] = string(f.Data)
			} else {
				filters[int(f.ID)] = f.FilePath
			}
		}
	}

	if len(s.conf.ParentalBlockHost) == 0 {
		s.conf.ParentalBlockHost = parentalBlockHost
	}
	if len(s.conf.SafeBrowsingBlockHost) == 0 {
		s.conf.SafeBrowsingBlockHost = safeBrowsingBlockHost
	}

	s.dnsFilter = dnsfilter.New(&s.conf.Config, filters)
	if s.dnsFilter == nil {
		return fmt.Errorf("could not initialize dnsfilter")
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
	return s.queryLog.flushLogBuffer(true)
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

// Return TRUE if this client should be blocked
func (s *Server) isBlockedIP(ip string) bool {
	if len(s.AllowedClients) != 0 || len(s.AllowedClientsIPNet) != 0 {
		_, ok := s.AllowedClients[ip]
		if ok {
			return false
		}

		if len(s.AllowedClientsIPNet) != 0 {
			ipAddr := net.ParseIP(ip)
			for _, ipnet := range s.AllowedClientsIPNet {
				if ipnet.Contains(ipAddr) {
					return false
				}
			}
		}

		return true
	}

	_, ok := s.DisallowedClients[ip]
	if ok {
		return true
	}

	if len(s.DisallowedClientsIPNet) != 0 {
		ipAddr := net.ParseIP(ip)
		for _, ipnet := range s.DisallowedClientsIPNet {
			if ipnet.Contains(ipAddr) {
				return true
			}
		}
	}

	return false
}

// Return TRUE if this domain should be blocked
func (s *Server) isBlockedDomain(host string) bool {
	_, ok := s.BlockedHosts[host]
	return ok
}

func (s *Server) beforeRequestHandler(p *proxy.Proxy, d *proxy.DNSContext) (bool, error) {
	ip, _, _ := net.SplitHostPort(d.Addr.String())
	if s.isBlockedIP(ip) {
		log.Tracef("Client IP %s is blocked by settings", ip)
		return false, nil
	}

	if len(d.Req.Question) == 1 {
		host := strings.TrimSuffix(d.Req.Question[0].Name, ".")
		if s.isBlockedDomain(host) {
			log.Tracef("Domain %s is blocked by settings", host)
			return false, nil
		}
	}

	return true, nil
}

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
func (s *Server) handleDNSRequest(p *proxy.Proxy, d *proxy.DNSContext) error {
	start := time.Now()

	if s.conf.OnDNSRequest != nil {
		s.conf.OnDNSRequest(d)
	}

	// use dnsfilter before cache -- changed settings or filters would require cache invalidation otherwise
	res, err := s.filterDNSRequest(d)
	if err != nil {
		return err
	}

	if d.Res == nil {
		answer := []dns.RR{}
		originalQuestion := d.Req.Question[0]

		if res.Reason == dnsfilter.ReasonRewrite && len(res.CanonName) != 0 {
			answer = append(answer, s.genCNAMEAnswer(d.Req, res.CanonName))
			// resolve canonical name, not the original host name
			d.Req.Question[0].Name = dns.Fqdn(res.CanonName)
		}

		// request was not filtered so let it be processed further
		err = p.Resolve(d)
		if err != nil {
			return err
		}

		if res.Reason == dnsfilter.ReasonRewrite && len(res.CanonName) != 0 {

			d.Req.Question[0] = originalQuestion
			d.Res.Question[0] = originalQuestion

			if len(d.Res.Answer) != 0 {
				answer = append(answer, d.Res.Answer...) // host -> IP
				d.Res.Answer = answer
			}
		}
	}

	shouldLog := true
	msg := d.Req

	// don't log ANY request if refuseAny is enabled
	if len(msg.Question) >= 1 && msg.Question[0].Qtype == dns.TypeANY && s.conf.RefuseAny {
		shouldLog = false
	}

	if s.conf.QueryLogEnabled && shouldLog {
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
	var res dnsfilter.Result
	req := d.Req
	host := strings.TrimSuffix(req.Question[0].Name, ".")
	origHost := host

	s.RLock()
	protectionEnabled := s.conf.ProtectionEnabled
	dnsFilter := s.dnsFilter
	s.RUnlock()

	if !protectionEnabled {
		return nil, nil
	}

	if host != origHost {
		log.Debug("Rewrite: not supported: CNAME for %s is %s", origHost, host)
	}

	var err error

	clientAddr := ""
	if d.Addr != nil {
		clientAddr, _, _ = net.SplitHostPort(d.Addr.String())
	}
	res, err = dnsFilter.CheckHost(host, d.Req.Question[0].Qtype, clientAddr)
	if err != nil {
		// Return immediately if there's an error
		return nil, errorx.Decorate(err, "dnsfilter failed to check host '%s'", host)

	} else if res.IsFiltered {
		// log.Tracef("Host %s is filtered, reason - '%s', matched rule: '%s'", host, res.Reason, res.Rule)
		d.Res = s.genDNSFilterMessage(d, &res)

	} else if res.Reason == dnsfilter.ReasonRewrite && len(res.IPList) != 0 {
		resp := dns.Msg{}
		resp.SetReply(req)

		name := host
		if len(res.CanonName) != 0 {
			resp.Answer = append(resp.Answer, s.genCNAMEAnswer(req, res.CanonName))
			name = res.CanonName
		}

		for _, ip := range res.IPList {
			if req.Question[0].Qtype == dns.TypeA {
				a := s.genAAnswer(req, ip)
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)

			} else if req.Question[0].Qtype == dns.TypeAAAA {
				a := s.genAAAAAnswer(req, res.IP)
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			}
		}

		d.Res = &resp
	}

	return &res, err
}

// genDNSFilterMessage generates a DNS message corresponding to the filtering result
func (s *Server) genDNSFilterMessage(d *proxy.DNSContext, result *dnsfilter.Result) *dns.Msg {
	m := d.Req

	if m.Question[0].Qtype != dns.TypeA && m.Question[0].Qtype != dns.TypeAAAA {
		return s.genNXDomain(m)
	}

	switch result.Reason {
	case dnsfilter.FilteredSafeBrowsing:
		return s.genBlockedHost(m, s.conf.SafeBrowsingBlockHost, d)
	case dnsfilter.FilteredParental:
		return s.genBlockedHost(m, s.conf.ParentalBlockHost, d)
	default:
		if result.IP != nil {
			return s.genResponseWithIP(m, result.IP)
		}

		if s.conf.BlockingMode == "null_ip" {
			switch m.Question[0].Qtype {
			case dns.TypeA:
				return s.genARecord(m, []byte{0, 0, 0, 0})
			case dns.TypeAAAA:
				return s.genAAAARecord(m, net.IPv6zero)
			}
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
	resp.Answer = append(resp.Answer, s.genAAnswer(request, ip))
	return &resp
}

func (s *Server) genAAAARecord(request *dns.Msg, ip net.IP) *dns.Msg {
	resp := dns.Msg{}
	resp.SetReply(request)
	resp.Answer = append(resp.Answer, s.genAAAAAnswer(request, ip))
	return &resp
}

func (s *Server) genAAnswer(req *dns.Msg, ip net.IP) *dns.A {
	answer := new(dns.A)
	answer.Hdr = dns.RR_Header{
		Name:   req.Question[0].Name,
		Rrtype: dns.TypeA,
		Ttl:    s.conf.BlockedResponseTTL,
		Class:  dns.ClassINET,
	}
	answer.A = ip
	return answer
}

func (s *Server) genAAAAAnswer(req *dns.Msg, ip net.IP) *dns.AAAA {
	answer := new(dns.AAAA)
	answer.Hdr = dns.RR_Header{
		Name:   req.Question[0].Name,
		Rrtype: dns.TypeAAAA,
		Ttl:    s.conf.BlockedResponseTTL,
		Class:  dns.ClassINET,
	}
	answer.AAAA = ip
	return answer
}

// generate DNS response message with an IP address
func (s *Server) genResponseWithIP(req *dns.Msg, ip net.IP) *dns.Msg {
	if req.Question[0].Qtype == dns.TypeA && ip.To4() != nil {
		return s.genARecord(req, ip.To4())
	} else if req.Question[0].Qtype == dns.TypeAAAA && ip.To4() == nil {
		return s.genAAAARecord(req, ip)
	}

	// empty response
	resp := dns.Msg{}
	resp.SetReply(req)
	return &resp
}

func (s *Server) genBlockedHost(request *dns.Msg, newAddr string, d *proxy.DNSContext) *dns.Msg {

	ip := net.ParseIP(newAddr)
	if ip != nil {
		return s.genResponseWithIP(request, ip)
	}

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

// Make a CNAME response
func (s *Server) genCNAMEAnswer(req *dns.Msg, cname string) *dns.CNAME {
	answer := new(dns.CNAME)
	answer.Hdr = dns.RR_Header{
		Name:   req.Question[0].Name,
		Rrtype: dns.TypeCNAME,
		Ttl:    s.conf.BlockedResponseTTL,
		Class:  dns.ClassINET,
	}
	answer.Target = dns.Fqdn(cname)
	return answer
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
			Ttl:    s.conf.BlockedResponseTTL,
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
