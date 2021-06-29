// Package dnsforward contains a DNS forwarding server.
package dnsforward

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// DefaultTimeout is the default upstream timeout
const DefaultTimeout = 10 * time.Second

// defaultClientIDCacheCount is the default count of items in the LRU client ID
// cache.  The assumption here is that there won't be more than this many
// requests between the BeforeRequestHandler stage and the actual processing.
const defaultClientIDCacheCount = 1024

const (
	safeBrowsingBlockHost = "standard-block.dns.adguard.com"
	parentalBlockHost     = "family-block.dns.adguard.com"
)

var defaultDNS = []string{
	"https://dns10.quad9.net/dns-query",
}
var defaultBootstrap = []string{"9.9.9.10", "149.112.112.10", "2620:fe::10", "2620:fe::fe:10"}

// Often requested by all kinds of DNS probes
var defaultBlockedHosts = []string{"version.bind", "id.server", "hostname.bind"}

var webRegistered bool

// hostToIPTable is an alias for the type of Server.tableHostToIP.
type hostToIPTable = map[string]net.IP

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
	dnsProxy   *proxy.Proxy          // DNS proxy instance
	dnsFilter  *filtering.DNSFilter  // DNS filter instance
	dhcpServer dhcpd.ServerInterface // DHCP server instance (optional)
	queryLog   querylog.QueryLog     // Query log instance
	stats      stats.Stats
	access     *accessCtx

	// localDomainSuffix is the suffix used to detect internal hosts.  It
	// must be a valid domain name plus dots on each side.
	localDomainSuffix string

	ipset          ipsetCtx
	subnetDetector *aghnet.SubnetDetector
	localResolvers *proxy.Proxy
	sysResolvers   aghnet.SystemResolvers
	recDetector    *recursionDetector

	tableHostToIP     hostToIPTable
	tableHostToIPLock sync.Mutex

	tableIPToHost     *aghnet.IPMap
	tableIPToHostLock sync.Mutex

	// clientIDCache is a temporary storage for clientIDs that were
	// extracted during the BeforeRequestHandler stage.
	clientIDCache cache.Cache

	// DNS proxy instance for internal usage
	// We don't Start() it and so no listen port is required.
	internalProxy *proxy.Proxy

	isRunning bool

	conf ServerConfig
	// serverLock protects Server.
	serverLock sync.RWMutex
}

// defaultLocalDomainSuffix is the default suffix used to detect internal hosts
// when no suffix is provided.
//
// See the documentation for Server.localDomainSuffix.
const defaultLocalDomainSuffix = ".lan."

// DNSCreateParams are parameters to create a new server.
type DNSCreateParams struct {
	DNSFilter      *filtering.DNSFilter
	Stats          stats.Stats
	QueryLog       querylog.QueryLog
	DHCPServer     dhcpd.ServerInterface
	SubnetDetector *aghnet.SubnetDetector
	LocalDomain    string
}

// domainNameToSuffix converts a domain name into a local domain suffix.
func domainNameToSuffix(tld string) (suffix string) {
	l := len(tld) + 2
	b := make([]byte, l)
	b[0] = '.'
	copy(b[1:], tld)
	b[l-1] = '.'

	return string(b)
}

const (
	// recursionTTL is the time recursive request is cached for.
	recursionTTL = 1 * time.Second
	// cachedRecurrentReqNum is the maximum number of cached recurrent
	// requests.
	cachedRecurrentReqNum = 1000
)

// NewServer creates a new instance of the dnsforward.Server
// Note: this function must be called only once
func NewServer(p DNSCreateParams) (s *Server, err error) {
	var localDomainSuffix string
	if p.LocalDomain == "" {
		localDomainSuffix = defaultLocalDomainSuffix
	} else {
		err = aghnet.ValidateDomainName(p.LocalDomain)
		if err != nil {
			return nil, fmt.Errorf("local domain: %w", err)
		}

		localDomainSuffix = domainNameToSuffix(p.LocalDomain)
	}

	s = &Server{
		dnsFilter:         p.DNSFilter,
		stats:             p.Stats,
		queryLog:          p.QueryLog,
		subnetDetector:    p.SubnetDetector,
		localDomainSuffix: localDomainSuffix,
		recDetector:       newRecursionDetector(recursionTTL, cachedRecurrentReqNum),
		clientIDCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  defaultClientIDCacheCount,
		}),
	}

	// TODO(e.burkov): Enable the refresher after the actual implementation
	// passes the public testing.
	s.sysResolvers, err = aghnet.NewSystemResolvers(0, nil)
	if err != nil {
		return nil, fmt.Errorf("initializing system resolvers: %w", err)
	}

	if p.DHCPServer != nil {
		s.dhcpServer = p.DHCPServer
		s.dhcpServer.SetOnLeaseChanged(s.onDHCPLeaseChanged)
		s.onDHCPLeaseChanged(dhcpd.LeaseChangedAdded)
	}

	if runtime.GOARCH == "mips" || runtime.GOARCH == "mipsle" {
		// Use plain DNS on MIPS, encryption is too slow
		defaultDNS = defaultBootstrap
	}

	return s, nil
}

// NewCustomServer creates a new instance of *Server with custom internal proxy.
func NewCustomServer(internalProxy *proxy.Proxy) *Server {
	s := &Server{
		recDetector: newRecursionDetector(0, 1),
	}
	if internalProxy != nil {
		s.internalProxy = internalProxy
	}

	return s
}

// Close gracefully closes the server.  It is safe for concurrent use.
//
// TODO(e.burkov): A better approach would be making Stop method waiting for all
// its workers finished.  But it would require the upstream.Upstream to have the
// Close method to prevent from hanging while waiting for unresponsive server to
// respond.
func (s *Server) Close() {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	s.dnsFilter = nil
	s.stats = nil
	s.queryLog = nil
	s.dnsProxy = nil

	if err := s.ipset.close(); err != nil {
		log.Error("closing ipset: %s", err)
	}
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *FilteringConfig) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	sc := s.conf.FilteringConfig
	*c = sc
	c.RatelimitWhitelist = aghstrings.CloneSlice(sc.RatelimitWhitelist)
	c.BootstrapDNS = aghstrings.CloneSlice(sc.BootstrapDNS)
	c.AllowedClients = aghstrings.CloneSlice(sc.AllowedClients)
	c.DisallowedClients = aghstrings.CloneSlice(sc.DisallowedClients)
	c.BlockedHosts = aghstrings.CloneSlice(sc.BlockedHosts)
	c.UpstreamDNS = aghstrings.CloneSlice(sc.UpstreamDNS)
}

// RDNSSettings returns the copy of actual RDNS configuration.
func (s *Server) RDNSSettings() (localPTRResolvers []string, resolveClients, resolvePTR bool) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return aghstrings.CloneSlice(s.conf.LocalPTRResolvers),
		s.conf.ResolveClients,
		s.conf.UsePrivateRDNS
}

// Resolve - get IP addresses by host name from an upstream server.
// No request/response filtering is performed.
// Query log and Stats are not updated.
// This method may be called before Start().
func (s *Server) Resolve(host string) ([]net.IPAddr, error) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return s.internalProxy.LookupIPAddr(host)
}

// RDNSExchanger is a resolver for clients' addresses.
type RDNSExchanger interface {
	// Exchange tries to resolve the ip in a suitable way, e.g. either as
	// local or as external.
	Exchange(ip net.IP) (host string, err error)
	// ResolvesPrivatePTR returns true if the RDNSExchanger is able to
	// resolve PTR requests for locally-served addresses.
	ResolvesPrivatePTR() (ok bool)
}

const (
	// rDNSEmptyAnswerErr is returned by Exchange method when the answer
	// section of respond is empty.
	rDNSEmptyAnswerErr errors.Error = "the answer section is empty"

	// rDNSNotPTRErr is returned by Exchange method when the response is not
	// of PTR type.
	rDNSNotPTRErr errors.Error = "the response is not a ptr"
)

// Exchange implements the RDNSExchanger interface for *Server.
func (s *Server) Exchange(ip net.IP) (host string, err error) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if !s.conf.ResolveClients {
		return "", nil
	}

	arpa := dns.Fqdn(aghnet.ReverseAddr(ip))
	req := &dns.Msg{
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
	ctx := &proxy.DNSContext{
		Proto:     "udp",
		Req:       req,
		StartTime: time.Now(),
	}

	resolver := s.internalProxy
	if s.subnetDetector.IsLocallyServedNetwork(ip) {
		if !s.conf.UsePrivateRDNS {
			return "", nil
		}

		resolver = s.localResolvers
		s.recDetector.add(*req)
	}

	if err = resolver.Resolve(ctx); err != nil {
		return "", err
	}

	resp := ctx.Res
	if len(resp.Answer) == 0 {
		return "", fmt.Errorf("lookup for %q: %w", arpa, rDNSEmptyAnswerErr)
	}

	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		return "", fmt.Errorf("type checking: %w", rDNSNotPTRErr)
	}

	return strings.TrimSuffix(ptr.Ptr, "."), nil
}

// ResolvesPrivatePTR implements the RDNSExchanger interface for *Server.
func (s *Server) ResolvesPrivatePTR() (ok bool) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return s.conf.UsePrivateRDNS
}

// Start starts the DNS server.
func (s *Server) Start() error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	return s.startLocked()
}

// startLocked starts the DNS server without locking. For internal use only.
func (s *Server) startLocked() error {
	err := s.dnsProxy.Start()
	if err == nil {
		s.isRunning = true
	}
	return err
}

// defaultLocalTimeout is the default timeout for resolving addresses from
// locally-served networks.  It is assumed that local resolvers should work much
// faster than ordinary upstreams.
const defaultLocalTimeout = 1 * time.Second

// collectDNSIPAddrs returns IP addresses the server is listening on without
// port numbersю  For internal use only.
func (s *Server) collectDNSIPAddrs() (addrs []string, err error) {
	addrs = make([]string, len(s.conf.TCPListenAddrs)+len(s.conf.UDPListenAddrs))
	var i int
	var ip net.IP
	for _, addr := range s.conf.TCPListenAddrs {
		if addr == nil {
			continue
		}

		if ip = addr.IP; ip.IsUnspecified() {
			return aghnet.CollectAllIfacesAddrs()
		}

		addrs[i] = ip.String()
		i++
	}
	for _, addr := range s.conf.UDPListenAddrs {
		if addr == nil {
			continue
		}

		if ip = addr.IP; ip.IsUnspecified() {
			return aghnet.CollectAllIfacesAddrs()
		}

		addrs[i] = ip.String()
		i++
	}

	return addrs[:i], nil
}

func (s *Server) filterOurDNSAddrs(addrs []string) (filtered []string, err error) {
	var ourAddrs []string
	ourAddrs, err = s.collectDNSIPAddrs()
	if err != nil {
		return nil, err
	}

	ourAddrsSet := aghstrings.NewSet(ourAddrs...)

	// TODO(e.burkov): The approach of subtracting sets of strings is not
	// really applicable here since in case of listening on all network
	// interfaces we should check the whole interface's network to cut off
	// all the loopback addresses as well.
	return aghstrings.FilterOut(addrs, ourAddrsSet.Has), nil
}

// setupResolvers initializes the resolvers for local addresses.  For internal
// use only.
func (s *Server) setupResolvers(localAddrs []string) (err error) {
	bootstraps := s.conf.BootstrapDNS
	if len(localAddrs) == 0 {
		localAddrs = s.sysResolvers.Get()
		bootstraps = nil
	}

	localAddrs, err = s.filterOurDNSAddrs(localAddrs)
	if err != nil {
		return err
	}

	log.Debug("upstreams to resolve PTR for local addresses: %v", localAddrs)

	var upsConfig *proxy.UpstreamConfig
	upsConfig, err = proxy.ParseUpstreamsConfig(
		localAddrs,
		&upstream.Options{
			Bootstrap: bootstraps,
			Timeout:   defaultLocalTimeout,
			// TODO(e.burkov): Should we verify server's ceritificates?
		},
	)
	if err != nil {
		return fmt.Errorf("parsing upstreams: %w", err)
	}

	s.localResolvers = &proxy.Proxy{
		Config: proxy.Config{
			UpstreamConfig: upsConfig,
		},
	}

	return nil
}

// Prepare the object
func (s *Server) Prepare(config *ServerConfig) error {
	// Initialize the server configuration
	// --
	if config != nil {
		s.conf = *config
		if s.conf.BlockingMode == "custom_ip" {
			if s.conf.BlockingIPv4 == nil || s.conf.BlockingIPv6 == nil {
				return fmt.Errorf("dns: invalid custom blocking IP address specified")
			}
		}
	}

	// Set default values in the case if nothing is configured
	// --
	s.initDefaultSettings()

	// Initialize ipset configuration
	// --
	err := s.ipset.init(s.conf.IpsetList)
	if err != nil {
		return err
	}

	log.Debug("inited ipset")

	// Prepare DNS servers settings
	// --
	err = s.prepareUpstreamSettings()
	if err != nil {
		return err
	}

	// Create DNS proxy configuration
	// --
	var proxyConfig proxy.Config
	proxyConfig, err = s.createProxyConfig()
	if err != nil {
		return err
	}

	// Prepare a DNS proxy instance that we use for internal DNS queries
	// --
	s.prepareIntlProxy()

	s.access, err = newAccessCtx(s.conf.AllowedClients, s.conf.DisallowedClients, s.conf.BlockedHosts)
	if err != nil {
		return err
	}

	// Register web handlers if necessary
	// --
	if !webRegistered && s.conf.HTTPRegister != nil {
		webRegistered = true
		s.registerHandlers()
	}

	// Create the main DNS proxy instance
	// --
	s.dnsProxy = &proxy.Proxy{Config: proxyConfig}

	err = s.setupResolvers(s.conf.LocalPTRResolvers)
	if err != nil {
		return fmt.Errorf("setting up resolvers: %w", err)
	}

	s.recDetector.clear()

	return nil
}

// Stop stops the DNS server.
func (s *Server) Stop() error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	return s.stopLocked()
}

// stopLocked stops the DNS server without locking. For internal use only.
func (s *Server) stopLocked() error {
	if s.dnsProxy != nil {
		err := s.dnsProxy.Stop()
		if err != nil {
			return fmt.Errorf("could not stop the DNS server properly: %w", err)
		}
	}

	s.isRunning = false
	return nil
}

// IsRunning returns true if the DNS server is running.
func (s *Server) IsRunning() bool {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return s.isRunning
}

// Reconfigure applies the new configuration to the DNS server.
func (s *Server) Reconfigure(config *ServerConfig) error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	log.Print("Start reconfiguring the server")
	err := s.stopLocked()
	if err != nil {
		return fmt.Errorf("could not reconfigure the server: %w", err)
	}

	// It seems that net.Listener.Close() doesn't close file descriptors right away.
	// We wait for some time and hope that this fd will be closed.
	time.Sleep(100 * time.Millisecond)

	err = s.Prepare(config)
	if err != nil {
		return fmt.Errorf("could not reconfigure the server: %w", err)
	}

	err = s.startLocked()
	if err != nil {
		return fmt.Errorf("could not reconfigure the server: %w", err)
	}

	return nil
}

// ServeHTTP is a HTTP handler method we use to provide DNS-over-HTTPS.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var p *proxy.Proxy

	func() {
		s.serverLock.RLock()
		defer s.serverLock.RUnlock()

		p = s.dnsProxy
	}()

	if p != nil {
		p.ServeHTTP(w, r)
	}
}

// IsBlockedClient returns true if the client is blocked by the current access
// settings.
func (s *Server) IsBlockedClient(ip net.IP, clientID string) (blocked bool, rule string) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	allowlistMode := s.access.allowlistMode()
	blockedByIP, rule := s.access.isBlockedIP(ip)
	blockedByClientID := s.access.isBlockedClientID(clientID)

	// Allow if at least one of the checks allows in allowlist mode, but
	// block if at least one of the checks blocks in blocklist mode.
	if allowlistMode && blockedByIP && blockedByClientID {
		log.Debug("client %s (id %q) is not in access allowlist", ip, clientID)

		// Return now without substituting the empty rule for the
		// clientID because the rule can't be empty here.
		return true, rule
	} else if !allowlistMode && (blockedByIP || blockedByClientID) {
		log.Debug("client %s (id %q) is in access blocklist", ip, clientID)

		blocked = true
	}

	if rule == "" {
		rule = clientID
	}

	return blocked, rule
}
