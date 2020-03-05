package dnsforward

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

// DefaultTimeout is the default upstream timeout
const DefaultTimeout = 10 * time.Second

const (
	safeBrowsingBlockHost = "standard-block.dns.adguard.com"
	parentalBlockHost     = "family-block.dns.adguard.com"
)

var defaultDNS = []string{
	"https://dns10.quad9.net/dns-query",
}
var defaultBootstrap = []string{"9.9.9.10", "149.112.112.10", "2620:fe::10", "2620:fe::fe:10"}

var webRegistered bool

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
	queryLog  querylog.QueryLog    // Query log instance
	stats     stats.Stats
	access    *accessCtx

	// DNS proxy instance for internal usage
	// We don't Start() it and so no listen port is required.
	internalProxy *proxy.Proxy

	isRunning bool

	sync.RWMutex
	conf ServerConfig
}

// NewServer creates a new instance of the dnsforward.Server
// Note: this function must be called only once
func NewServer(dnsFilter *dnsfilter.Dnsfilter, stats stats.Stats, queryLog querylog.QueryLog) *Server {
	s := &Server{}
	s.dnsFilter = dnsFilter
	s.stats = stats
	s.queryLog = queryLog

	if runtime.GOARCH == "mips" || runtime.GOARCH == "mipsle" {
		// Use plain DNS on MIPS, encryption is too slow
		defaultDNS = defaultBootstrap
	}
	return s
}

// Close - close object
func (s *Server) Close() {
	s.Lock()
	s.dnsFilter = nil
	s.stats = nil
	s.queryLog = nil
	s.dnsProxy = nil
	s.Unlock()
}

func stringArrayDup(a []string) []string {
	a2 := make([]string, len(a))
	copy(a2, a)
	return a2
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *FilteringConfig) {
	s.RLock()
	sc := s.conf.FilteringConfig
	*c = sc
	c.RatelimitWhitelist = stringArrayDup(sc.RatelimitWhitelist)
	c.BootstrapDNS = stringArrayDup(sc.BootstrapDNS)
	c.AllowedClients = stringArrayDup(sc.AllowedClients)
	c.DisallowedClients = stringArrayDup(sc.DisallowedClients)
	c.BlockedHosts = stringArrayDup(sc.BlockedHosts)
	c.UpstreamDNS = stringArrayDup(sc.UpstreamDNS)
	s.RUnlock()
}

// FilteringConfig represents the DNS filtering configuration of AdGuard Home
// The zero FilteringConfig is empty and ready for use.
type FilteringConfig struct {
	// Filtering callback function
	FilterHandler func(clientAddr string, settings *dnsfilter.RequestFilteringSettings) `yaml:"-"`

	// This callback function returns the list of upstream servers for a client specified by IP address
	GetUpstreamsByClient func(clientAddr string) []upstream.Upstream `yaml:"-"`

	ProtectionEnabled bool `yaml:"protection_enabled"` // whether or not use any of dnsfilter features

	BlockingMode     string `yaml:"blocking_mode"` // mode how to answer filtered requests
	BlockingIPv4     string `yaml:"blocking_ipv4"` // IP address to be returned for a blocked A request
	BlockingIPv6     string `yaml:"blocking_ipv6"` // IP address to be returned for a blocked AAAA request
	BlockingIPAddrv4 net.IP `yaml:"-"`
	BlockingIPAddrv6 net.IP `yaml:"-"`

	BlockedResponseTTL uint32   `yaml:"blocked_response_ttl"` // if 0, then default is used (3600)
	Ratelimit          uint32   `yaml:"ratelimit"`            // max number of requests per second from a given IP (0 to disable)
	RatelimitWhitelist []string `yaml:"ratelimit_whitelist"`  // a list of whitelisted client IP addresses
	RefuseAny          bool     `yaml:"refuse_any"`           // if true, refuse ANY requests
	BootstrapDNS       []string `yaml:"bootstrap_dns"`        // a list of bootstrap DNS for DoH and DoT (plain DNS only)
	AllServers         bool     `yaml:"all_servers"`          // if true, parallel queries to all configured upstream servers are enabled

	EnableEDNSClientSubnet bool `yaml:"edns_client_subnet"` // Enable EDNS Client Subnet option

	// Respond with an empty answer to all AAAA requests
	AAAADisabled bool `yaml:"aaaa_disabled"`

	AllowedClients    []string `yaml:"allowed_clients"`    // IP addresses of whitelist clients
	DisallowedClients []string `yaml:"disallowed_clients"` // IP addresses of clients that should be blocked
	BlockedHosts      []string `yaml:"blocked_hosts"`      // hosts that should be blocked

	// IP (or domain name) which is used to respond to DNS requests blocked by parental control or safe-browsing
	ParentalBlockHost     string `yaml:"parental_block_host"`
	SafeBrowsingBlockHost string `yaml:"safebrowsing_block_host"`

	CacheSize   uint     `yaml:"cache_size"` // DNS cache size (in bytes)
	UpstreamDNS []string `yaml:"upstream_dns"`
}

// TLSConfig is the TLS configuration for HTTPS, DNS-over-HTTPS, and DNS-over-TLS
type TLSConfig struct {
	TLSListenAddr    *net.TCPAddr `yaml:"-" json:"-"`
	StrictSNICheck   bool         `yaml:"strict_sni_check" json:"-"`                  // Reject connection if the client uses server name (in SNI) that doesn't match the certificate
	CertificateChain string       `yaml:"certificate_chain" json:"certificate_chain"` // PEM-encoded certificates chain
	PrivateKey       string       `yaml:"private_key" json:"private_key"`             // PEM-encoded private key

	CertificatePath string `yaml:"certificate_path" json:"certificate_path"` // certificate file name
	PrivateKeyPath  string `yaml:"private_key_path" json:"private_key_path"` // private key file name

	CertificateChainData []byte `yaml:"-" json:"-"`
	PrivateKeyData       []byte `yaml:"-" json:"-"`

	cert     tls.Certificate // nolint(structcheck) - linter thinks that this field is unused, while TLSConfig is directly included into ServerConfig
	dnsNames []string        // nolint(structcheck) // DNS names from certificate (SAN) or CN value from Subject
}

// ServerConfig represents server configuration.
// The zero ServerConfig is empty and ready for use.
type ServerConfig struct {
	UDPListenAddr            *net.UDPAddr                   // UDP listen address
	TCPListenAddr            *net.TCPAddr                   // TCP listen address
	Upstreams                []upstream.Upstream            // Configured upstreams
	DomainsReservedUpstreams map[string][]upstream.Upstream // Map of domains and lists of configured upstreams
	OnDNSRequest             func(d *proxy.DNSContext)

	FilteringConfig
	TLSConfig

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))
}

// if any of ServerConfig values are zero, then default values from below are used
var defaultValues = ServerConfig{
	UDPListenAddr:   &net.UDPAddr{Port: 53},
	TCPListenAddr:   &net.TCPAddr{Port: 53},
	FilteringConfig: FilteringConfig{BlockedResponseTTL: 3600},
}

// Resolve - get IP addresses by host name from an upstream server.
// No request/response filtering is performed.
// Query log and Stats are not updated.
// This method may be called before Start().
func (s *Server) Resolve(host string) ([]net.IPAddr, error) {
	s.RLock()
	defer s.RUnlock()
	return s.internalProxy.LookupIPAddr(host)
}

// Exchange - send DNS request to an upstream server and receive response
// No request/response filtering is performed.
// Query log and Stats are not updated.
// This method may be called before Start().
func (s *Server) Exchange(req *dns.Msg) (*dns.Msg, error) {
	s.RLock()
	defer s.RUnlock()

	ctx := &proxy.DNSContext{
		Proto:     "udp",
		Req:       req,
		StartTime: time.Now(),
	}
	err := s.internalProxy.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	return ctx.Res, nil
}

// Start starts the DNS server
func (s *Server) Start() error {
	s.Lock()
	defer s.Unlock()
	return s.startInternal()
}

// startInternal starts without locking
func (s *Server) startInternal() error {
	err := s.dnsProxy.Start()
	if err == nil {
		s.isRunning = true
	}
	return err
}

// Prepare the object
// nolint(gocyclo)
func (s *Server) Prepare(config *ServerConfig) error {
	if config != nil {
		s.conf = *config
		if s.conf.BlockingMode == "custom_ip" {
			s.conf.BlockingIPAddrv4 = net.ParseIP(s.conf.BlockingIPv4)
			s.conf.BlockingIPAddrv6 = net.ParseIP(s.conf.BlockingIPv6)
			if s.conf.BlockingIPAddrv4 == nil || s.conf.BlockingIPAddrv6 == nil {
				return fmt.Errorf("DNS: invalid custom blocking IP address specified")
			}
		}
	}

	if len(s.conf.UpstreamDNS) == 0 {
		s.conf.UpstreamDNS = defaultDNS
	}
	if len(s.conf.BootstrapDNS) == 0 {
		s.conf.BootstrapDNS = defaultBootstrap
	}

	upstreamConfig, err := proxy.ParseUpstreamsConfig(s.conf.UpstreamDNS, s.conf.BootstrapDNS, DefaultTimeout)
	if err != nil {
		return fmt.Errorf("DNS: proxy.ParseUpstreamsConfig: %s", err)
	}
	s.conf.Upstreams = upstreamConfig.Upstreams
	s.conf.DomainsReservedUpstreams = upstreamConfig.DomainReservedUpstreams

	if len(s.conf.ParentalBlockHost) == 0 {
		s.conf.ParentalBlockHost = parentalBlockHost
	}
	if len(s.conf.SafeBrowsingBlockHost) == 0 {
		s.conf.SafeBrowsingBlockHost = safeBrowsingBlockHost
	}
	if s.conf.UDPListenAddr == nil {
		s.conf.UDPListenAddr = defaultValues.UDPListenAddr
	}
	if s.conf.TCPListenAddr == nil {
		s.conf.TCPListenAddr = defaultValues.TCPListenAddr
	}

	proxyConfig := proxy.Config{
		UDPListenAddr:            s.conf.UDPListenAddr,
		TCPListenAddr:            s.conf.TCPListenAddr,
		Ratelimit:                int(s.conf.Ratelimit),
		RatelimitWhitelist:       s.conf.RatelimitWhitelist,
		RefuseAny:                s.conf.RefuseAny,
		CacheEnabled:             true,
		CacheSizeBytes:           int(s.conf.CacheSize),
		Upstreams:                s.conf.Upstreams,
		DomainsReservedUpstreams: s.conf.DomainsReservedUpstreams,
		BeforeRequestHandler:     s.beforeRequestHandler,
		RequestHandler:           s.handleDNSRequest,
		AllServers:               s.conf.AllServers,
		EnableEDNSClientSubnet:   s.conf.EnableEDNSClientSubnet,
	}

	intlProxyConfig := proxy.Config{
		CacheEnabled:             true,
		CacheSizeBytes:           4096,
		Upstreams:                s.conf.Upstreams,
		DomainsReservedUpstreams: s.conf.DomainsReservedUpstreams,
	}
	s.internalProxy = &proxy.Proxy{Config: intlProxyConfig}

	s.access = &accessCtx{}
	err = s.access.Init(s.conf.AllowedClients, s.conf.DisallowedClients, s.conf.BlockedHosts)
	if err != nil {
		return err
	}

	if s.conf.TLSListenAddr != nil && len(s.conf.CertificateChainData) != 0 && len(s.conf.PrivateKeyData) != 0 {
		proxyConfig.TLSListenAddr = s.conf.TLSListenAddr
		s.conf.cert, err = tls.X509KeyPair(s.conf.CertificateChainData, s.conf.PrivateKeyData)
		if err != nil {
			return errorx.Decorate(err, "Failed to parse TLS keypair")
		}

		if s.conf.StrictSNICheck {
			x, err := x509.ParseCertificate(s.conf.cert.Certificate[0])
			if err != nil {
				return errorx.Decorate(err, "x509.ParseCertificate(): %s", err)
			}
			if len(x.DNSNames) != 0 {
				s.conf.dnsNames = x.DNSNames
				log.Debug("DNS: using DNS names from certificate's SAN: %v", x.DNSNames)
				sort.Strings(s.conf.dnsNames)
			} else {
				s.conf.dnsNames = append(s.conf.dnsNames, x.Subject.CommonName)
				log.Debug("DNS: using DNS name from certificate's CN: %s", x.Subject.CommonName)
			}
		}

		proxyConfig.TLSConfig = &tls.Config{
			GetCertificate: s.onGetCertificate,
			MinVersion:     tls.VersionTLS12,
		}
	}

	if len(proxyConfig.Upstreams) == 0 {
		log.Fatal("len(proxyConfig.Upstreams) == 0")
	}

	if !webRegistered && s.conf.HTTPRegister != nil {
		webRegistered = true
		s.registerHandlers()
	}

	// Initialize and start the DNS proxy
	s.dnsProxy = &proxy.Proxy{Config: proxyConfig}
	return nil
}

// Find value in a sorted array
func findSorted(ar []string, val string) int {
	i := sort.SearchStrings(ar, val)
	if i == len(ar) || ar[i] != val {
		return -1
	}
	return i
}

func isWildcard(host string) bool {
	return len(host) >= 2 &&
		host[0] == '*' && host[1] == '.'
}

// Return TRUE if host name matches a wildcard pattern
func matchDomainWildcard(host, wildcard string) bool {
	return isWildcard(wildcard) &&
		strings.HasSuffix(host, wildcard[1:])
}

// Return TRUE if client's SNI value matches DNS names from certificate
func matchDNSName(dnsNames []string, sni string) bool {
	if utils.IsValidHostname(sni) != nil {
		return false
	}
	if findSorted(dnsNames, sni) != -1 {
		return true
	}

	for _, dn := range dnsNames {
		if matchDomainWildcard(sni, dn) {
			return true
		}
	}
	return false
}

// Called by 'tls' package when Client Hello is received
// If the server name (from SNI) supplied by client is incorrect - we terminate the ongoing TLS handshake.
func (s *Server) onGetCertificate(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if s.conf.StrictSNICheck && !matchDNSName(s.conf.dnsNames, ch.ServerName) {
		log.Info("DNS: TLS: unknown SNI in Client Hello: %s", ch.ServerName)
		return nil, fmt.Errorf("Invalid SNI")
	}
	return &s.conf.cert, nil
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
		if err != nil {
			return errorx.Decorate(err, "could not stop the DNS server properly")
		}
	}

	s.isRunning = false
	return nil
}

// IsRunning returns true if the DNS server is running
func (s *Server) IsRunning() bool {
	s.RLock()
	defer s.RUnlock()
	return s.isRunning
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

	// It seems that net.Listener.Close() doesn't close file descriptors right away.
	// We wait for some time and hope that this fd will be closed.
	time.Sleep(100 * time.Millisecond)

	err = s.Prepare(config)
	if err != nil {
		return errorx.Decorate(err, "could not reconfigure the server")
	}

	err = s.startInternal()
	if err != nil {
		return errorx.Decorate(err, "could not reconfigure the server")
	}

	return nil
}

// ServeHTTP is a HTTP handler method we use to provide DNS-over-HTTPS
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	p := s.dnsProxy
	s.RUnlock()
	if p != nil { // an attempt to protect against race in case we're here after Close() was called
		p.ServeHTTP(w, r)
	}
}

// Get IP address from net.Addr object
// Note: we can't use net.SplitHostPort(a.String()) because of IPv6 zone:
// https://github.com/AdguardTeam/AdGuardHome/issues/1261
func ipFromAddr(a net.Addr) string {
	switch addr := a.(type) {
	case *net.UDPAddr:
		return addr.IP.String()
	case *net.TCPAddr:
		return addr.IP.String()
	}
	return ""
}

func (s *Server) beforeRequestHandler(p *proxy.Proxy, d *proxy.DNSContext) (bool, error) {
	ip := ipFromAddr(d.Addr)
	if s.access.IsBlockedIP(ip) {
		log.Tracef("Client IP %s is blocked by settings", ip)
		return false, nil
	}

	if len(d.Req.Question) == 1 {
		host := strings.TrimSuffix(d.Req.Question[0].Name, ".")
		if s.access.IsBlockedDomain(host) {
			log.Tracef("Domain %s is blocked by settings", host)
			return false, nil
		}
	}

	return true, nil
}

// To transfer information between modules
type dnsContext struct {
	srv                  *Server
	proxyCtx             *proxy.DNSContext
	setts                *dnsfilter.RequestFilteringSettings // filtering settings for this client
	startTime            time.Time
	result               *dnsfilter.Result
	origResp             *dns.Msg     // response received from upstream servers.  Set when response is modified by filtering
	origQuestion         dns.Question // question received from client.  Set when Rewrites are used.
	err                  error        // error returned from the module
	protectionEnabled    bool         // filtering is enabled, dnsfilter object is ready
	responseFromUpstream bool         // response is received from upstream servers
}

const (
	resultDone   = iota // module has completed its job, continue
	resultFinish        // module has completed its job, exit normally
	resultError         // an error occurred, exit with an error
)

// Perform initial checks;  process WHOIS & rDNS
func processInitial(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx
	if s.conf.AAAADisabled && d.Req.Question[0].Qtype == dns.TypeAAAA {
		_ = proxy.CheckDisabledAAAARequest(d, true)
		return resultFinish
	}

	if s.conf.OnDNSRequest != nil {
		s.conf.OnDNSRequest(d)
	}

	// disable Mozilla DoH
	if (d.Req.Question[0].Qtype == dns.TypeA || d.Req.Question[0].Qtype == dns.TypeAAAA) &&
		d.Req.Question[0].Name == "use-application-dns.net." {
		d.Res = s.genNXDomain(d.Req)
		return resultFinish
	}

	return resultDone
}

// Apply filtering logic
func processFilteringBeforeRequest(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx

	s.RLock()
	// Synchronize access to s.dnsFilter so it won't be suddenly uninitialized while in use.
	// This could happen after proxy server has been stopped, but its workers are not yet exited.
	//
	// A better approach is for proxy.Stop() to wait until all its workers exit,
	//  but this would require the Upstream interface to have Close() function
	//  (to prevent from hanging while waiting for unresponsive DNS server to respond).

	var err error
	ctx.protectionEnabled = s.conf.ProtectionEnabled && s.dnsFilter != nil
	if ctx.protectionEnabled {
		ctx.setts = s.getClientRequestFilteringSettings(d)
		ctx.result, err = s.filterDNSRequest(ctx)
	}
	s.RUnlock()

	if err != nil {
		ctx.err = err
		return resultError
	}
	return resultDone
}

// Pass request to upstream servers;  process the response
func processUpstream(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx
	if d.Res != nil {
		return resultDone // response is already set - nothing to do
	}

	if d.Addr != nil && s.conf.GetUpstreamsByClient != nil {
		clientIP := ipFromAddr(d.Addr)
		upstreams := s.conf.GetUpstreamsByClient(clientIP)
		if len(upstreams) > 0 {
			log.Debug("Using custom upstreams for %s", clientIP)
			d.Upstreams = upstreams
		}
	}

	// request was not filtered so let it be processed further
	err := s.dnsProxy.Resolve(d)
	if err != nil {
		ctx.err = err
		return resultError
	}

	ctx.responseFromUpstream = true
	return resultDone
}

// Apply filtering logic after we have received response from upstream servers
func processFilteringAfterResponse(ctx *dnsContext) int {
	s := ctx.srv
	d := ctx.proxyCtx
	res := ctx.result
	var err error

	if !ctx.responseFromUpstream {
		return resultDone // don't process response if it's not from upstream servers
	}

	if res.Reason == dnsfilter.ReasonRewrite && len(res.CanonName) != 0 {
		d.Req.Question[0] = ctx.origQuestion
		d.Res.Question[0] = ctx.origQuestion

		if len(d.Res.Answer) != 0 {
			answer := []dns.RR{}
			answer = append(answer, s.genCNAMEAnswer(d.Req, res.CanonName))
			answer = append(answer, d.Res.Answer...) // host -> IP
			d.Res.Answer = answer
		}

	} else if res.Reason != dnsfilter.NotFilteredWhiteList && ctx.protectionEnabled {
		origResp2 := d.Res
		ctx.result, err = s.filterDNSResponse(ctx)
		if err != nil {
			ctx.err = err
			return resultError
		}
		if ctx.result != nil {
			ctx.origResp = origResp2 // matched by response
		} else {
			ctx.result = &dnsfilter.Result{}
		}
	}

	return resultDone
}

// Write Stats data and logs
func processQueryLogsAndStats(ctx *dnsContext) int {
	elapsed := time.Since(ctx.startTime)
	s := ctx.srv
	d := ctx.proxyCtx

	shouldLog := true
	msg := d.Req

	// don't log ANY request if refuseAny is enabled
	if len(msg.Question) >= 1 && msg.Question[0].Qtype == dns.TypeANY && s.conf.RefuseAny {
		shouldLog = false
	}

	s.RLock()
	// Synchronize access to s.queryLog and s.stats so they won't be suddenly uninitialized while in use.
	// This can happen after proxy server has been stopped, but its workers haven't yet exited.
	if shouldLog && s.queryLog != nil {
		p := querylog.AddParams{
			Question:   msg,
			Answer:     d.Res,
			OrigAnswer: ctx.origResp,
			Result:     ctx.result,
			Elapsed:    elapsed,
			ClientIP:   getIP(d.Addr),
		}
		if d.Upstream != nil {
			p.Upstream = d.Upstream.Address()
		}
		s.queryLog.Add(p)
	}

	s.updateStats(d, elapsed, *ctx.result)
	s.RUnlock()

	return resultDone
}

// handleDNSRequest filters the incoming DNS requests and writes them to the query log
// nolint (gocyclo)
func (s *Server) handleDNSRequest(p *proxy.Proxy, d *proxy.DNSContext) error {
	ctx := &dnsContext{srv: s, proxyCtx: d}
	ctx.result = &dnsfilter.Result{}
	ctx.startTime = time.Now()

	type modProcessFunc func(ctx *dnsContext) int
	mods := []modProcessFunc{
		processInitial,
		processFilteringBeforeRequest,
		processUpstream,
		processFilteringAfterResponse,
		processQueryLogsAndStats,
	}
	for _, process := range mods {
		r := process(ctx)
		switch r {
		case resultFinish:
			return nil
		case resultError:
			return ctx.err
		}
	}

	if d.Res != nil {
		d.Res.Compress = true // some devices require DNS message compression
	}
	return nil
}

// Get IP address from net.Addr
func getIP(addr net.Addr) net.IP {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP
	case *net.TCPAddr:
		return addr.IP
	}
	return nil
}

func (s *Server) updateStats(d *proxy.DNSContext, elapsed time.Duration, res dnsfilter.Result) {
	if s.stats == nil {
		return
	}

	e := stats.Entry{}
	e.Domain = strings.ToLower(d.Req.Question[0].Name)
	e.Domain = e.Domain[:len(e.Domain)-1] // remove last "."
	switch addr := d.Addr.(type) {
	case *net.UDPAddr:
		e.Client = addr.IP
	case *net.TCPAddr:
		e.Client = addr.IP
	}
	e.Time = uint32(elapsed / 1000)
	switch res.Reason {

	case dnsfilter.NotFilteredNotFound:
		fallthrough
	case dnsfilter.NotFilteredWhiteList:
		fallthrough
	case dnsfilter.NotFilteredError:
		e.Result = stats.RNotFiltered

	case dnsfilter.FilteredSafeBrowsing:
		e.Result = stats.RSafeBrowsing
	case dnsfilter.FilteredParental:
		e.Result = stats.RParental
	case dnsfilter.FilteredSafeSearch:
		e.Result = stats.RSafeSearch

	case dnsfilter.FilteredBlackList:
		fallthrough
	case dnsfilter.FilteredInvalid:
		fallthrough
	case dnsfilter.FilteredBlockedService:
		e.Result = stats.RFiltered
	}
	s.stats.Update(e)
}

// getClientRequestFilteringSettings lookups client filtering settings
// using the client's IP address from the DNSContext
func (s *Server) getClientRequestFilteringSettings(d *proxy.DNSContext) *dnsfilter.RequestFilteringSettings {
	setts := s.dnsFilter.GetConfig()
	setts.FilteringEnabled = true
	if s.conf.FilterHandler != nil {
		clientAddr := ipFromAddr(d.Addr)
		s.conf.FilterHandler(clientAddr, &setts)
	}
	return &setts
}

// filterDNSRequest applies the dnsFilter and sets d.Res if the request was filtered
func (s *Server) filterDNSRequest(ctx *dnsContext) (*dnsfilter.Result, error) {
	d := ctx.proxyCtx
	req := d.Req
	host := strings.TrimSuffix(req.Question[0].Name, ".")
	res, err := s.dnsFilter.CheckHost(host, d.Req.Question[0].Qtype, ctx.setts)
	if err != nil {
		// Return immediately if there's an error
		return nil, errorx.Decorate(err, "dnsfilter failed to check host '%s'", host)

	} else if res.IsFiltered {
		// log.Tracef("Host %s is filtered, reason - '%s', matched rule: '%s'", host, res.Reason, res.Rule)
		d.Res = s.genDNSFilterMessage(d, &res)

	} else if res.Reason == dnsfilter.ReasonRewrite && len(res.IPList) != 0 {
		resp := s.makeResponse(req)

		name := host
		if len(res.CanonName) != 0 {
			resp.Answer = append(resp.Answer, s.genCNAMEAnswer(req, res.CanonName))
			name = res.CanonName
		}

		for _, ip := range res.IPList {
			ip4 := ip.To4()
			if req.Question[0].Qtype == dns.TypeA && ip4 != nil {
				a := s.genAAnswer(req, ip4)
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			} else if req.Question[0].Qtype == dns.TypeAAAA && ip4 == nil {
				a := s.genAAAAAnswer(req, ip)
				a.Hdr.Name = dns.Fqdn(name)
				resp.Answer = append(resp.Answer, a)
			}
		}

		d.Res = resp

	} else if res.Reason == dnsfilter.ReasonRewrite && len(res.CanonName) != 0 {
		ctx.origQuestion = d.Req.Question[0]
		// resolve canonical name, not the original host name
		d.Req.Question[0].Name = dns.Fqdn(res.CanonName)
	}

	return &res, err
}

// If response contains CNAME, A or AAAA records, we apply filtering to each canonical host name or IP address.
// If this is a match, we set a new response in d.Res and return.
func (s *Server) filterDNSResponse(ctx *dnsContext) (*dnsfilter.Result, error) {
	d := ctx.proxyCtx
	for _, a := range d.Res.Answer {
		host := ""

		switch v := a.(type) {
		case *dns.CNAME:
			log.Debug("DNSFwd: Checking CNAME %s for %s", v.Target, v.Hdr.Name)
			host = strings.TrimSuffix(v.Target, ".")

		case *dns.A:
			host = v.A.String()
			log.Debug("DNSFwd: Checking record A (%s) for %s", host, v.Hdr.Name)

		case *dns.AAAA:
			host = v.AAAA.String()
			log.Debug("DNSFwd: Checking record AAAA (%s) for %s", host, v.Hdr.Name)

		default:
			continue
		}

		s.RLock()
		// Synchronize access to s.dnsFilter so it won't be suddenly uninitialized while in use.
		// This could happen after proxy server has been stopped, but its workers are not yet exited.
		if !s.conf.ProtectionEnabled || s.dnsFilter == nil {
			s.RUnlock()
			continue
		}
		res, err := s.dnsFilter.CheckHostRules(host, d.Req.Question[0].Qtype, ctx.setts)
		s.RUnlock()

		if err != nil {
			return nil, err

		} else if res.IsFiltered {
			d.Res = s.genDNSFilterMessage(d, &res)
			log.Debug("DNSFwd: Matched %s by response: %s", d.Req.Question[0].Name, host)
			return &res, nil
		}
	}

	return nil, nil
}

// Create a DNS response by DNS request and set necessary flags
func (s *Server) makeResponse(req *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetReply(req)
	resp.RecursionAvailable = true
	resp.Compress = true
	return &resp
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
		// If the query was filtered by "Safe search", dnsfilter also must return
		// the IP address that must be used in response.
		// In this case regardless of the filtering method, we should return it
		if result.Reason == dnsfilter.FilteredSafeSearch && result.IP != nil {
			return s.genResponseWithIP(m, result.IP)
		}

		if s.conf.BlockingMode == "null_ip" {
			// it means that we should return 0.0.0.0 or :: for any blocked request

			switch m.Question[0].Qtype {
			case dns.TypeA:
				return s.genARecord(m, []byte{0, 0, 0, 0})
			case dns.TypeAAAA:
				return s.genAAAARecord(m, net.IPv6zero)
			}

		} else if s.conf.BlockingMode == "custom_ip" {
			// means that we should return custom IP for any blocked request

			switch m.Question[0].Qtype {
			case dns.TypeA:
				return s.genARecord(m, s.conf.BlockingIPAddrv4)
			case dns.TypeAAAA:
				return s.genAAAARecord(m, s.conf.BlockingIPAddrv6)
			}

		} else if s.conf.BlockingMode == "nxdomain" {
			// means that we should return NXDOMAIN for any blocked request

			return s.genNXDomain(m)
		}

		// Default blocking mode
		// If there's an IP specified in the rule, return it
		// If there is no IP, return NXDOMAIN
		if result.IP != nil {
			return s.genResponseWithIP(m, result.IP)
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
	resp := s.makeResponse(request)
	resp.Answer = append(resp.Answer, s.genAAnswer(request, ip))
	return resp
}

func (s *Server) genAAAARecord(request *dns.Msg, ip net.IP) *dns.Msg {
	resp := s.makeResponse(request)
	resp.Answer = append(resp.Answer, s.genAAAAAnswer(request, ip))
	return resp
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
	} else if req.Question[0].Qtype == dns.TypeAAAA &&
		len(ip) == net.IPv6len && ip.To4() == nil {
		return s.genAAAARecord(req, ip)
	}

	// empty response
	resp := s.makeResponse(req)
	return resp
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

	resp := s.makeResponse(request)
	if newContext.Res != nil {
		for _, answer := range newContext.Res.Answer {
			answer.Header().Name = request.Question[0].Name
			resp.Answer = append(resp.Answer, answer)
		}
	}

	return resp
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
