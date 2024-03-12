// Package dnsforward contains a DNS forwarding server.
package dnsforward

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/sysresolv"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
)

// DefaultTimeout is the default upstream timeout
const DefaultTimeout = 10 * time.Second

// defaultLocalTimeout is the default timeout for resolving addresses from
// locally-served networks.  It is assumed that local resolvers should work much
// faster than ordinary upstreams.
const defaultLocalTimeout = 1 * time.Second

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

var (
	// defaultUDPListenAddrs are the default UDP addresses for the server.
	defaultUDPListenAddrs = []*net.UDPAddr{{Port: 53}}

	// defaultTCPListenAddrs are the default TCP addresses for the server.
	defaultTCPListenAddrs = []*net.TCPAddr{{Port: 53}}
)

var webRegistered bool

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

// SystemResolvers is an interface for accessing the OS-provided resolvers.
type SystemResolvers interface {
	// Addrs returns the list of system resolvers' addresses.  Callers must
	// clone the returned slice before modifying it.  Implementations of Addrs
	// must be safe for concurrent use.
	Addrs() (addrs []netip.AddrPort)
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
	// dnsProxy is the DNS proxy for forwarding client's DNS requests.
	dnsProxy *proxy.Proxy

	// dnsFilter is the DNS filter for filtering client's DNS requests and
	// responses.
	dnsFilter *filtering.DNSFilter

	// dhcpServer is the DHCP server for accessing lease data.
	dhcpServer DHCP

	// queryLog is the query log for client's DNS requests, responses and
	// filtering results.
	queryLog querylog.QueryLog

	// stats is the statistics collector for client's DNS usage data.
	stats stats.Interface

	// access drops disallowed clients.
	access *accessManager

	// localDomainSuffix is the suffix used to detect internal hosts.  It
	// must be a valid domain name plus dots on each side.
	localDomainSuffix string

	// ipset processes DNS requests using ipset data.
	ipset ipsetCtx

	// privateNets is the configured set of IP networks considered private.
	privateNets netutil.SubnetSet

	// addrProc, if not nil, is used to process clients' IP addresses with rDNS,
	// WHOIS, etc.
	addrProc client.AddressProcessor

	// localResolvers is a DNS proxy instance used to resolve PTR records for
	// addresses considered private as per the [privateNets].
	//
	// TODO(e.burkov):  Remove once the local resolvers logic moved to dnsproxy.
	localResolvers *proxy.Proxy

	// sysResolvers used to fetch system resolvers to use by default for private
	// PTR resolving.
	sysResolvers SystemResolvers

	// etcHosts contains the current data from the system's hosts files.
	etcHosts upstream.Resolver

	// bootstrap is the resolver for upstreams' hostnames.
	bootstrap upstream.Resolver

	// bootResolvers are the resolvers that should be used for
	// bootstrapping along with [etcHosts].
	//
	// TODO(e.burkov):  Use [proxy.UpstreamConfig] when it will implement the
	// [upstream.Resolver] interface.
	bootResolvers []*upstream.UpstreamResolver

	// recDetector is a cache for recursive requests.  It is used to detect and
	// prevent recursive requests only for private upstreams.
	//
	// See https://github.com/adguardTeam/adGuardHome/issues/3185#issuecomment-851048135.
	recDetector *recursionDetector

	// dns64Pref is the NAT64 prefix used for DNS64 response mapping.  The major
	// part of DNS64 happens inside the [proxy] package, but there still are
	// some places where response mapping is needed (e.g. DHCP).
	dns64Pref netip.Prefix

	// anonymizer masks the client's IP addresses if needed.
	anonymizer *aghnet.IPMut

	// clientIDCache is a temporary storage for ClientIDs that were extracted
	// during the BeforeRequestHandler stage.
	clientIDCache cache.Cache

	// internalProxy resolves internal requests from the application itself.  It
	// isn't started and so no listen ports are required.
	internalProxy *proxy.Proxy

	// isRunning is true if the DNS server is running.
	isRunning bool

	// protectionUpdateInProgress is used to make sure that only one goroutine
	// updating the protection configuration after a pause is running at a time.
	protectionUpdateInProgress atomic.Bool

	// conf is the current configuration of the server.
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
	DHCPServer  DHCP
	PrivateNets netutil.SubnetSet
	Anonymizer  *aghnet.IPMut
	EtcHosts    *aghnet.HostsContainer
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
//
// TODO(a.garipov): How many constructors and initializers does this thing have?
// Refactor!
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

	var etcHosts upstream.Resolver
	if p.EtcHosts != nil {
		etcHosts = upstream.NewHostsResolver(p.EtcHosts)
	}

	s = &Server{
		dnsFilter:   p.DNSFilter,
		dhcpServer:  p.DHCPServer,
		stats:       p.Stats,
		queryLog:    p.QueryLog,
		privateNets: p.PrivateNets,
		// TODO(e.burkov):  Use some case-insensitive string comparison.
		localDomainSuffix: strings.ToLower(localDomainSuffix),
		etcHosts:          etcHosts,
		recDetector:       newRecursionDetector(recursionTTL, cachedRecurrentReqNum),
		clientIDCache: cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  defaultClientIDCacheCount,
		}),
		anonymizer: p.Anonymizer,
		conf: ServerConfig{
			ServePlainDNS: true,
		},
	}

	s.sysResolvers, err = sysresolv.NewSystemResolvers(nil, defaultPlainDNSPort)
	if err != nil {
		return nil, fmt.Errorf("initializing system resolvers: %w", err)
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

	// TODO(s.chzhen):  Remove it.
	s.stats = nil
	s.queryLog = nil
	s.dnsProxy = nil

	if err := s.ipset.close(); err != nil {
		log.Error("dnsforward: closing ipset: %s", err)
	}
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *Config) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	sc := s.conf.Config
	*c = sc
	c.RatelimitWhitelist = slices.Clone(sc.RatelimitWhitelist)
	c.BootstrapDNS = stringutil.CloneSlice(sc.BootstrapDNS)
	c.FallbackDNS = stringutil.CloneSlice(sc.FallbackDNS)
	c.AllowedClients = stringutil.CloneSlice(sc.AllowedClients)
	c.DisallowedClients = stringutil.CloneSlice(sc.DisallowedClients)
	c.BlockedHosts = stringutil.CloneSlice(sc.BlockedHosts)
	c.TrustedProxies = slices.Clone(sc.TrustedProxies)
	c.UpstreamDNS = stringutil.CloneSlice(sc.UpstreamDNS)
}

// LocalPTRResolvers returns the current local PTR resolver configuration.
func (s *Server) LocalPTRResolvers() (localPTRResolvers []string) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return stringutil.CloneSlice(s.conf.LocalPTRResolvers)
}

// AddrProcConfig returns the current address processing configuration.  Only
// fields c.UsePrivateRDNS, c.UseRDNS, and c.UseWHOIS are filled.
func (s *Server) AddrProcConfig() (c *client.DefaultAddrProcConfig) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return &client.DefaultAddrProcConfig{
		UsePrivateRDNS: s.conf.UsePrivateRDNS,
		UseRDNS:        s.conf.AddrProcConf.UseRDNS,
		UseWHOIS:       s.conf.AddrProcConf.UseWHOIS,
	}
}

// Resolve gets IP addresses by host name from an upstream server.  No
// request/response filtering is performed.  Query log and Stats are not
// updated.  This method may be called before [Server.Start].
func (s *Server) Resolve(ctx context.Context, net, host string) (addr []netip.Addr, err error) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return s.internalProxy.LookupNetIP(ctx, net, host)
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
var _ rdns.Exchanger = (*Server)(nil)

// Exchange implements the [rdns.Exchanger] interface for *Server.
func (s *Server) Exchange(ip netip.Addr) (host string, ttl time.Duration, err error) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	arpa, err := netutil.IPToReversedAddr(ip.AsSlice())
	if err != nil {
		return "", 0, fmt.Errorf("reversing ip: %w", err)
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

	dctx := &proxy.DNSContext{
		Proto: "udp",
		Req:   req,
	}

	var resolver *proxy.Proxy
	var errMsg string
	if s.privateNets.Contains(ip) {
		if !s.conf.UsePrivateRDNS {
			return "", 0, nil
		}

		resolver = s.localResolvers
		errMsg = "resolving a private address: %w"
		s.recDetector.add(*req)
	} else {
		resolver = s.internalProxy
		errMsg = "resolving an address: %w"
	}
	if err = resolver.Resolve(dctx); err != nil {
		return "", 0, fmt.Errorf(errMsg, err)
	}

	return hostFromPTR(dctx.Res)
}

// hostFromPTR returns domain name from the PTR response or error.
func hostFromPTR(resp *dns.Msg) (host string, ttl time.Duration, err error) {
	// Distinguish between NODATA response and a failed request.
	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		return "", 0, fmt.Errorf(
			"received %s response: %w",
			dns.RcodeToString[resp.Rcode],
			ErrRDNSFailed,
		)
	}

	var ttlSec uint32

	log.Debug("dnsforward: resolving ptr, received %d answers", len(resp.Answer))
	for _, ans := range resp.Answer {
		ptr, ok := ans.(*dns.PTR)
		if !ok {
			continue
		}

		// Respect zero TTL records since some DNS servers use it to
		// locally-resolved addresses.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/6046.
		if ptr.Hdr.Ttl >= ttlSec {
			host = ptr.Ptr
			ttlSec = ptr.Hdr.Ttl
		}
	}

	if host != "" {
		// NOTE:  Don't use [aghnet.NormalizeDomain] to retain original letter
		// case.
		host = strings.TrimSuffix(host, ".")
		ttl = time.Duration(ttlSec) * time.Second

		return host, ttl, nil
	}

	return "", 0, ErrRDNSNoData
}

// Start starts the DNS server.  It must only be called after [Server.Prepare].
func (s *Server) Start() error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	return s.startLocked()
}

// startLocked starts the DNS server without locking.  s.serverLock is expected
// to be locked.
func (s *Server) startLocked() error {
	// TODO(e.burkov):  Use context properly.
	err := s.dnsProxy.Start(context.Background())
	if err == nil {
		s.isRunning = true
	}

	return err
}

// prepareLocalResolvers initializes the local upstreams configuration using
// boot as bootstrap.  It assumes that s.serverLock is locked or s not running.
func (s *Server) prepareLocalResolvers(
	boot upstream.Resolver,
) (uc *proxy.UpstreamConfig, err error) {
	set, err := s.conf.ourAddrsSet()
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return nil, err
	}

	resolvers := s.conf.LocalPTRResolvers
	confNeedsFiltering := len(resolvers) > 0
	if confNeedsFiltering {
		resolvers = stringutil.FilterOut(resolvers, IsCommentOrEmpty)
	} else {
		sysResolvers := slices.DeleteFunc(slices.Clone(s.sysResolvers.Addrs()), set.Has)
		resolvers = make([]string, 0, len(sysResolvers))
		for _, r := range sysResolvers {
			resolvers = append(resolvers, r.String())
		}
	}

	log.Debug("dnsforward: upstreams to resolve ptr for local addresses: %v", resolvers)

	uc, err = s.prepareUpstreamConfig(resolvers, nil, &upstream.Options{
		Bootstrap: boot,
		Timeout:   defaultLocalTimeout,
		// TODO(e.burkov): Should we verify server's certificates?
		PreferIPv6: s.conf.BootstrapPreferIPv6,
	})
	if err != nil {
		return nil, fmt.Errorf("preparing private upstreams: %w", err)
	}

	if confNeedsFiltering {
		err = filterOutAddrs(uc, set)
		if err != nil {
			return nil, fmt.Errorf("filtering private upstreams: %w", err)
		}
	}

	return uc, nil
}

// setupLocalResolvers initializes and sets the resolvers for local addresses.
// It assumes s.serverLock is locked or s not running.  It returns the upstream
// configuration used for private PTR resolving, or nil if it's disabled.  Note,
// that it's safe to put nil into [proxy.Config.PrivateRDNSUpstreamConfig].
func (s *Server) setupLocalResolvers(boot upstream.Resolver) (uc *proxy.UpstreamConfig, err error) {
	if !s.conf.UsePrivateRDNS {
		// It's safe to put nil into [proxy.Config.PrivateRDNSUpstreamConfig].
		return nil, nil
	}

	uc, err = s.prepareLocalResolvers(boot)
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return nil, err
	}

	s.localResolvers, err = proxy.New(&proxy.Config{
		UpstreamConfig: uc,
	})
	if err != nil {
		return nil, fmt.Errorf("creating local resolvers: %w", err)
	}

	// TODO(e.burkov):  Should we also consider the DNS64 usage?
	return uc, nil
}

// Prepare initializes parameters of s using data from conf.  conf must not be
// nil.
func (s *Server) Prepare(conf *ServerConfig) (err error) {
	s.conf = *conf

	// dnsFilter can be nil during application update.
	if s.dnsFilter != nil {
		mode, bIPv4, bIPv6 := s.dnsFilter.BlockingMode()
		err = validateBlockingMode(mode, bIPv4, bIPv6)
		if err != nil {
			return fmt.Errorf("checking blocking mode: %w", err)
		}
	}

	s.initDefaultSettings()

	boot, err := s.prepareInternalDNS()
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	proxyConfig, err := s.newProxyConfig()
	if err != nil {
		return fmt.Errorf("preparing proxy: %w", err)
	}

	s.setupDNS64()

	s.access, err = newAccessCtx(
		s.conf.AllowedClients,
		s.conf.DisallowedClients,
		s.conf.BlockedHosts,
	)
	if err != nil {
		return fmt.Errorf("preparing access: %w", err)
	}

	// TODO(e.burkov):  Remove once the local resolvers logic moved to dnsproxy.
	proxyConfig.PrivateRDNSUpstreamConfig, err = s.setupLocalResolvers(boot)
	if err != nil {
		return fmt.Errorf("setting up resolvers: %w", err)
	}

	proxyConfig.Fallbacks, err = s.setupFallbackDNS()
	if err != nil {
		return fmt.Errorf("setting up fallback dns servers: %w", err)
	}

	s.dnsProxy, err = proxy.New(proxyConfig)
	if err != nil {
		return fmt.Errorf("creating proxy: %w", err)
	}

	s.recDetector.clear()

	s.setupAddrProc()

	s.registerHandlers()

	return nil
}

// prepareInternalDNS initializes the internal state of s before initializing
// the primary DNS proxy instance.  It assumes s.serverLock is locked or the
// Server not running.
func (s *Server) prepareInternalDNS() (boot upstream.Resolver, err error) {
	err = s.prepareIpsetListSettings()
	if err != nil {
		return nil, fmt.Errorf("preparing ipset settings: %w", err)
	}

	s.bootstrap, s.bootResolvers, err = s.createBootstrap(s.conf.BootstrapDNS, &upstream.Options{
		Timeout:      DefaultTimeout,
		HTTPVersions: UpstreamHTTPVersions(s.conf.UseHTTP3Upstreams),
	})
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	err = s.prepareUpstreamSettings(s.bootstrap)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return s.bootstrap, err
	}

	err = s.prepareInternalProxy()
	if err != nil {
		return s.bootstrap, fmt.Errorf("preparing internal proxy: %w", err)
	}

	return s.bootstrap, nil
}

// setupFallbackDNS initializes the fallback DNS servers.
func (s *Server) setupFallbackDNS() (uc *proxy.UpstreamConfig, err error) {
	fallbacks := s.conf.FallbackDNS
	fallbacks = stringutil.FilterOut(fallbacks, IsCommentOrEmpty)
	if len(fallbacks) == 0 {
		return nil, nil
	}

	uc, err = proxy.ParseUpstreamsConfig(fallbacks, &upstream.Options{
		// TODO(s.chzhen):  Investigate if other options are needed.
		Timeout:    s.conf.UpstreamTimeout,
		PreferIPv6: s.conf.BootstrapPreferIPv6,
		// TODO(e.burkov):  Use bootstrap.
	})
	if err != nil {
		// Do not wrap the error because it's informative enough as is.
		return nil, err
	}

	return uc, nil
}

// setupAddrProc initializes the address processor.  It assumes s.serverLock is
// locked or the Server not running.
func (s *Server) setupAddrProc() {
	// TODO(a.garipov): This is a crutch for tests; remove.
	if s.conf.AddrProcConf == nil {
		s.conf.AddrProcConf = &client.DefaultAddrProcConfig{}
	}
	if s.conf.AddrProcConf.AddressUpdater == nil {
		s.addrProc = client.EmptyAddrProc{}
	} else {
		c := s.conf.AddrProcConf
		c.DialContext = s.DialContext
		c.PrivateSubnets = s.privateNets
		c.UsePrivateRDNS = s.conf.UsePrivateRDNS
		s.addrProc = client.NewDefaultAddrProc(s.conf.AddrProcConf)

		// Clear the initial addresses to not resolve them again.
		//
		// TODO(a.garipov): Consider ways of removing this once more client
		// logic is moved to package client.
		c.InitialAddresses = nil
	}
}

// validateBlockingMode returns an error if the blocking mode data aren't valid.
func validateBlockingMode(
	mode filtering.BlockingMode,
	blockingIPv4, blockingIPv6 netip.Addr,
) (err error) {
	switch mode {
	case
		filtering.BlockingModeDefault,
		filtering.BlockingModeNXDOMAIN,
		filtering.BlockingModeREFUSED,
		filtering.BlockingModeNullIP:
		return nil
	case filtering.BlockingModeCustomIP:
		if !blockingIPv4.Is4() {
			return fmt.Errorf("blocking_ipv4 must be valid ipv4 on custom_ip blocking_mode")
		} else if !blockingIPv6.Is6() {
			return fmt.Errorf("blocking_ipv6 must be valid ipv6 on custom_ip blocking_mode")
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
		MaxGoroutines:  s.conf.MaxGoroutines,
	}

	err = setProxyUpstreamMode(conf, srvConf.UpstreamMode, srvConf.FastestTimeout.Duration)
	if err != nil {
		return fmt.Errorf("invalid upstream mode: %w", err)
	}

	s.internalProxy, err = proxy.New(conf)

	return err
}

// Stop stops the DNS server.
func (s *Server) Stop() error {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	return s.stopLocked()
}

// stopLocked stops the DNS server without locking.  s.serverLock is expected to
// be locked.
func (s *Server) stopLocked() (err error) {
	// TODO(e.burkov, a.garipov):  Return critical errors, not just log them.
	// This will require filtering all the non-critical errors in
	// [upstream.Upstream] implementations.

	if s.dnsProxy != nil {
		// TODO(e.burkov):  Use context properly.
		err = s.dnsProxy.Shutdown(context.Background())
		if err != nil {
			log.Error("dnsforward: closing primary resolvers: %s", err)
		}
	}

	logCloserErr(s.internalProxy.UpstreamConfig, "dnsforward: closing internal resolvers: %s")
	if s.localResolvers != nil {
		logCloserErr(s.localResolvers.UpstreamConfig, "dnsforward: closing local resolvers: %s")
	}

	for _, b := range s.bootResolvers {
		logCloserErr(b, "dnsforward: closing bootstrap %s: %s", b.Address())
	}

	s.isRunning = false

	return nil
}

// logCloserErr logs the error returned by c, if any.
func logCloserErr(c io.Closer, format string, args ...any) {
	if c == nil {
		return
	}

	err := c.Close()
	if err != nil {
		log.Error(format, append(args, err)...)
	}
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
	} else {
		closeErr := s.addrProc.Close()
		if closeErr != nil {
			log.Error("dnsforward: closing address processor: %s", closeErr)
		}
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
