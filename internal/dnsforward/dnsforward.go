// Package dnsforward contains a DNS forwarding server.
package dnsforward

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
)

// DefaultTimeout is the default upstream timeout
const DefaultTimeout = 10 * time.Second

// defaultClientIDCacheCount is the default count of items in the LRU ClientID
// cache.  The assumption here is that there won't be more than this many
// requests between the BeforeRequestHandler stage and the actual processing.
const defaultClientIDCacheCount = 1024

var defaultDNS = []string{
	"https://dns10.quad9.net/dns-query",
}
var defaultBootstrap = []string{"9.9.9.10", "149.112.112.10", "2620:fe::10", "2620:fe::fe:10"}

// Often requested by all kinds of DNS probes
var defaultBlockedHosts = []string{"version.bind", "id.server", "hostname.bind"}

var webRegistered bool

// hostToIPTable is a convenient type alias for tables of host names to an IP
// address.
//
// TODO(e.burkov):  Use the [DHCP] interface instead.
type hostToIPTable = map[string]netip.Addr

// ipToHostTable is a convenient type alias for tables of IP addresses to their
// host names.  For example, for use with PTR queries.
//
// TODO(e.burkov):  Use the [DHCP] interface instead.
type ipToHostTable = map[netip.Addr]string

// DHCP is an interface for accessing DHCP lease data needed in this package.
type DHCP interface {
	// HostByIP returns the hostname of the DHCP client with the given IP
	// address.  The address will be netip.Addr{} if there is no such client,
	// due to an assumption that a DHCP client must always have an IP address.
	HostByIP(ip netip.Addr) (host string)

	// IPByHost returns the IP address of the DHCP client with the given
	// hostname.  The hostname will be an empty string if there is no such
	// client, due to an assumption that a DHCP client must always have a
	// hostname, either set by the client or assigned automatically.
	IPByHost(host string) (ip netip.Addr)

	// Enabled returns true if DHCP provides information about clients.
	Enabled() (ok bool)
}

// Server is the main way to start a DNS server.
//
// Example:
//
//	s := dnsforward.Server{}
//	err := s.Start(nil) // will start a DNS server listening on default port 53, in a goroutine
//	err := s.Reconfigure(ServerConfig{UDPListenAddr: &net.UDPAddr{Port: 53535}}) // will reconfigure running DNS server to listen on UDP port 53535
//	err := s.Stop() // will stop listening on port 53535 and cancel all goroutines
//	err := s.Start(nil) // will start listening again, on port 53535, in a goroutine
//
// The zero Server is empty and ready for use.
type Server struct {
	dnsProxy   *proxy.Proxy         // DNS proxy instance
	dnsFilter  *filtering.DNSFilter // DNS filter instance
	dhcpServer dhcpd.Interface      // DHCP server instance (optional)
	queryLog   querylog.QueryLog    // Query log instance
	stats      stats.Interface
	access     *accessManager

	// localDomainSuffix is the suffix used to detect internal hosts.  It
	// must be a valid domain name plus dots on each side.
	localDomainSuffix string

	ipset          ipsetCtx
	privateNets    netutil.SubnetSet
	localResolvers *proxy.Proxy
	sysResolvers   aghnet.SystemResolvers

	// recDetector is a cache for recursive requests.  It is used to detect
	// and prevent recursive requests only for private upstreams.
	//
	// See https://github.com/adguardTeam/adGuardHome/issues/3185#issuecomment-851048135.
	recDetector *recursionDetector

	// dns64Pref is the NAT64 prefix used for DNS64 response mapping.  The major
	// part of DNS64 happens inside the [proxy] package, but there still are
	// some places where response mapping is needed (e.g. DHCP).
	dns64Pref netip.Prefix

	// anonymizer masks the client's IP addresses if needed.
	anonymizer *aghnet.IPMut

	tableHostToIP     hostToIPTable
	tableHostToIPLock sync.Mutex

	tableIPToHost     ipToHostTable
	tableIPToHostLock sync.Mutex

	// clientIDCache is a temporary storage for ClientIDs that were extracted
	// during the BeforeRequestHandler stage.
	clientIDCache cache.Cache

	// DNS proxy instance for internal usage
	// We don't Start() it and so no listen port is required.
	internalProxy *proxy.Proxy

	isRunning bool

	// protectionUpdateInProgress is used to make sure that only one goroutine
	// updating the protection configuration after a pause is running at a time.
	protectionUpdateInProgress atomic.Bool

	conf ServerConfig
	// serverLock protects Server.
	serverLock sync.RWMutex
}

// defaultLocalDomainSuffix is the default suffix used to detect internal hosts
// when no suffix is provided.
//
// See the documentation for Server.localDomainSuffix.
const defaultLocalDomainSuffix = "lan"

// DNSCreateParams are parameters to create a new server.
type DNSCreateParams struct {
	DNSFilter   *filtering.DNSFilter
	Stats       stats.Interface
	QueryLog    querylog.QueryLog
	DHCPServer  dhcpd.Interface
	PrivateNets netutil.SubnetSet
	Anonymizer  *aghnet.IPMut
	LocalDomain string
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
		err = netutil.ValidateDomainName(p.LocalDomain)
		if err != nil {
			return nil, fmt.Errorf("local domain: %w", err)
		}

		localDomainSuffix = p.LocalDomain
	}

	if p.Anonymizer == nil {
		p.Anonymizer = aghnet.NewIPMut(nil)
	}
	s = &Server{
		dnsFilter:         p.DNSFilter,
		stats:             p.Stats,
		queryLog:          p.QueryLog,
		privateNets:       p.PrivateNets,
		localDomainSuffix: localDomainSuffix,
		recDetector:       newRecursionDetector(recursionTTL, cachedRecurrentReqNum),
		clientIDCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  defaultClientIDCacheCount,
		}),
		anonymizer: p.Anonymizer,
	}

	// TODO(e.burkov): Enable the refresher after the actual implementation
	// passes the public testing.
	s.sysResolvers, err = aghnet.NewSystemResolvers(nil)
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
		log.Error("dnsforward: closing ipset: %s", err)
	}
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *FilteringConfig) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	sc := s.conf.FilteringConfig
	*c = sc
	c.RatelimitWhitelist = stringutil.CloneSlice(sc.RatelimitWhitelist)
	c.BootstrapDNS = stringutil.CloneSlice(sc.BootstrapDNS)
	c.AllowedClients = stringutil.CloneSlice(sc.AllowedClients)
	c.DisallowedClients = stringutil.CloneSlice(sc.DisallowedClients)
	c.BlockedHosts = stringutil.CloneSlice(sc.BlockedHosts)
	c.TrustedProxies = stringutil.CloneSlice(sc.TrustedProxies)
	c.UpstreamDNS = stringutil.CloneSlice(sc.UpstreamDNS)
}

// RDNSSettings returns the copy of actual RDNS configuration.
func (s *Server) RDNSSettings() (localPTRResolvers []string, resolveClients, resolvePTR bool) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return stringutil.CloneSlice(s.conf.LocalPTRResolvers),
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
	// Exchange tries to resolve the ip in a suitable way, i.e. either as local
	// or as external.
	Exchange(ip net.IP) (host string, err error)

	// ResolvesPrivatePTR returns true if the RDNSExchanger is able to
	// resolve PTR requests for locally-served addresses.
	ResolvesPrivatePTR() (ok bool)
}

const (
	// ErrRDNSNoData is returned by [RDNSExchanger.Exchange] when the answer
	// section of response is either NODATA or has no PTR records.
	ErrRDNSNoData errors.Error = "no ptr data in response"

	// ErrRDNSFailed is returned by [RDNSExchanger.Exchange] if the received
	// response is not a NOERROR or NXDOMAIN.
	ErrRDNSFailed errors.Error = "failed to resolve ptr"
)

// type check
var _ RDNSExchanger = (*Server)(nil)

// Exchange implements the RDNSExchanger interface for *Server.
func (s *Server) Exchange(ip net.IP) (host string, err error) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	if !s.conf.ResolveClients {
		return "", nil
	}

	arpa, err := netutil.IPToReversedAddr(ip)
	if err != nil {
		return "", fmt.Errorf("reversing ip: %w", err)
	}

	arpa = dns.Fqdn(arpa)
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

	var resolver *proxy.Proxy
	if s.privateNets.Contains(ip) {
		if !s.conf.UsePrivateRDNS {
			return "", nil
		}

		resolver = s.localResolvers
		s.recDetector.add(*req)
	} else {
		resolver = s.internalProxy
	}

	if err = resolver.Resolve(ctx); err != nil {
		return "", err
	}

	// Distinguish between NODATA response and a failed request.
	resp := ctx.Res
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		return "", fmt.Errorf(
			"received %s response: %w",
			dns.RcodeToString[resp.Rcode],
			ErrRDNSFailed,
		)
	}

	for _, ans := range resp.Answer {
		ptr, ok := ans.(*dns.PTR)
		if ok {
			return strings.TrimSuffix(ptr.Ptr, "."), nil
		}
	}

	return "", ErrRDNSNoData
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
// port numbers.  For internal use only.
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

	ourAddrsSet := stringutil.NewSet(ourAddrs...)

	// TODO(e.burkov): The approach of subtracting sets of strings is not
	// really applicable here since in case of listening on all network
	// interfaces we should check the whole interface's network to cut off
	// all the loopback addresses as well.
	return stringutil.FilterOut(addrs, ourAddrsSet.Has), nil
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

	log.Debug("dnsforward: upstreams to resolve ptr for local addresses: %v", localAddrs)

	upsConfig, err := s.prepareUpstreamConfig(localAddrs, nil, &upstream.Options{
		Bootstrap: bootstraps,
		Timeout:   defaultLocalTimeout,
		// TODO(e.burkov): Should we verify server's certificates?

		PreferIPv6: s.conf.BootstrapPreferIPv6,
	})
	if err != nil {
		return fmt.Errorf("parsing private upstreams: %w", err)
	}

	s.localResolvers = &proxy.Proxy{
		Config: proxy.Config{
			UpstreamConfig: upsConfig,
		},
	}

	return nil
}

// Prepare initializes parameters of s using data from conf.  conf must not be
// nil.
func (s *Server) Prepare(conf *ServerConfig) (err error) {
	s.conf = *conf

	err = validateBlockingMode(s.conf.BlockingMode, s.conf.BlockingIPv4, s.conf.BlockingIPv6)
	if err != nil {
		return fmt.Errorf("checking blocking mode: %w", err)
	}

	s.initDefaultSettings()

	err = s.prepareIpsetListSettings()
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return fmt.Errorf("preparing ipset settings: %w", err)
	}

	err = s.prepareUpstreamSettings()
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	var proxyConfig proxy.Config
	proxyConfig, err = s.createProxyConfig()
	if err != nil {
		return fmt.Errorf("preparing proxy: %w", err)
	}

	s.setupDNS64()

	err = s.prepareInternalProxy()
	if err != nil {
		return fmt.Errorf("preparing internal proxy: %w", err)
	}

	s.access, err = newAccessCtx(
		s.conf.AllowedClients,
		s.conf.DisallowedClients,
		s.conf.BlockedHosts,
	)
	if err != nil {
		return fmt.Errorf("preparing access: %w", err)
	}

	s.registerHandlers()

	// TODO(e.burkov):  Remove once the local resolvers logic moved to dnsproxy.
	err = s.setupResolvers(s.conf.LocalPTRResolvers)
	if err != nil {
		return fmt.Errorf("setting up resolvers: %w", err)
	}

	if s.conf.UsePrivateRDNS {
		proxyConfig.PrivateRDNSUpstreamConfig = s.localResolvers.UpstreamConfig
	}

	s.dnsProxy = &proxy.Proxy{Config: proxyConfig}

	s.recDetector.clear()

	return nil
}

// validateBlockingMode returns an error if the blocking mode data aren't valid.
func validateBlockingMode(mode BlockingMode, blockingIPv4, blockingIPv6 net.IP) (err error) {
	switch mode {
	case
		BlockingModeDefault,
		BlockingModeNXDOMAIN,
		BlockingModeREFUSED,
		BlockingModeNullIP:
		return nil
	case BlockingModeCustomIP:
		if blockingIPv4 == nil {
			return fmt.Errorf("blocking_ipv4 must be set when blocking_mode is custom_ip")
		} else if blockingIPv6 == nil {
			return fmt.Errorf("blocking_ipv6 must be set when blocking_mode is custom_ip")
		}

		return nil
	default:
		return fmt.Errorf("bad blocking mode %q", mode)
	}
}

// prepareInternalProxy initializes the DNS proxy that is used for internal DNS
// queries, such as public clients PTR resolving and updater hostname resolving.
func (s *Server) prepareInternalProxy() (err error) {
	srvConf := s.conf
	conf := &proxy.Config{
		CacheEnabled:   true,
		CacheSizeBytes: 4096,
		UpstreamConfig: srvConf.UpstreamConfig,
		MaxGoroutines:  int(s.conf.MaxGoroutines),
	}

	setProxyUpstreamMode(
		conf,
		srvConf.AllServers,
		srvConf.FastestAddr,
		srvConf.FastestTimeout.Duration,
	)

	// TODO(a.garipov): Make a proper constructor for proxy.Proxy.
	p := &proxy.Proxy{
		Config: *conf,
	}

	err = p.Init()
	if err != nil {
		return err
	}

	s.internalProxy = p

	return nil
}

// Stop stops the DNS server.
func (s *Server) Stop() error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	return s.stopLocked()
}

// stopLocked stops the DNS server without locking.  For internal use only.
func (s *Server) stopLocked() (err error) {
	// TODO(e.burkov, a.garipov):  Return critical errors, not just log them.
	// This will require filtering all the non-critical errors in
	// [upstream.Upstream] implementations.

	if s.dnsProxy != nil {
		err = s.dnsProxy.Stop()
		if err != nil {
			log.Error("dnsforward: closing primary resolvers: %s", err)
		}
	}

	if upsConf := s.internalProxy.UpstreamConfig; upsConf != nil {
		err = upsConf.Close()
		if err != nil {
			log.Error("dnsforward: closing internal resolvers: %s", err)
		}
	}

	if upsConf := s.localResolvers.UpstreamConfig; upsConf != nil {
		err = upsConf.Close()
		if err != nil {
			log.Error("dnsforward: closing local resolvers: %s", err)
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

// srvClosedErr is returned when the method can't complete without inaccessible
// data from the closing server.
const srvClosedErr errors.Error = "server is closed"

// proxy returns a pointer to the current DNS proxy instance.  If p is nil, the
// server is closing.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/3655.
func (s *Server) proxy() (p *proxy.Proxy) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return s.dnsProxy
}

// Reconfigure applies the new configuration to the DNS server.
func (s *Server) Reconfigure(conf *ServerConfig) error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	log.Info("dnsforward: starting reconfiguring server")
	defer log.Info("dnsforward: finished reconfiguring server")

	err := s.stopLocked()
	if err != nil {
		return fmt.Errorf("could not reconfigure the server: %w", err)
	}

	// It seems that net.Listener.Close() doesn't close file descriptors right away.
	// We wait for some time and hope that this fd will be closed.
	time.Sleep(100 * time.Millisecond)

	// TODO(a.garipov): This whole piece of API is weird and needs to be remade.
	if conf == nil {
		conf = &s.conf
	}

	err = s.Prepare(conf)
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
	if prx := s.proxy(); prx != nil {
		prx.ServeHTTP(w, r)
	}
}

// IsBlockedClient returns true if the client is blocked by the current access
// settings.
func (s *Server) IsBlockedClient(ip netip.Addr, clientID string) (blocked bool, rule string) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	blockedByIP := false
	if ip != (netip.Addr{}) {
		blockedByIP, rule = s.access.isBlockedIP(ip)
	}

	allowlistMode := s.access.allowlistMode()
	blockedByClientID := s.access.isBlockedClientID(clientID)

	// Allow if at least one of the checks allows in allowlist mode, but block
	// if at least one of the checks blocks in blocklist mode.
	if allowlistMode && blockedByIP && blockedByClientID {
		log.Debug("dnsforward: client %v (id %q) is not in access allowlist", ip, clientID)

		// Return now without substituting the empty rule for the
		// clientID because the rule can't be empty here.
		return true, rule
	} else if !allowlistMode && (blockedByIP || blockedByClientID) {
		log.Debug("dnsforward: client %v (id %q) is in access blocklist", ip, clientID)

		blocked = true
	}

	return blocked, aghalg.Coalesce(rule, clientID)
}
