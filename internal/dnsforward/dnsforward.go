// Package dnsforward contains a DNS forwarding server.
package dnsforward

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
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

// Often requested by all kinds of DNS probes
var defaultBlockedHosts = []string{"version.bind", "id.server", "hostname.bind"}

var webRegistered bool

// hostToIPTable is an alias for the type of Server.tableHostToIP.
type hostToIPTable = map[string]net.IP

// ipToHostTable is an alias for the type of Server.tableIPToHost.
//
// TODO(a.garipov): Define an IPMap type in aghnet and use here and in other
// places?
type ipToHostTable = map[string]string

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
	dnsFilter  *dnsfilter.DNSFilter  // DNS filter instance
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

	tableHostToIP     hostToIPTable
	tableHostToIPLock sync.Mutex

	tableIPToHost     ipToHostTable
	tableIPToHostLock sync.Mutex

	// DNS proxy instance for internal usage
	// We don't Start() it and so no listen port is required.
	internalProxy *proxy.Proxy

	isRunning bool

	sync.RWMutex
	conf ServerConfig
}

// defaultLocalDomainSuffix is the default suffix used to detect internal hosts
// when no suffix is provided.
//
// See the documentation for Server.localDomainSuffix.
const defaultLocalDomainSuffix = ".lan."

// DNSCreateParams are parameters to create a new server.
type DNSCreateParams struct {
	DNSFilter      *dnsfilter.DNSFilter
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
	s := &Server{}
	if internalProxy != nil {
		s.internalProxy = internalProxy
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

	err := s.ipset.Close()
	if err != nil {
		log.Error("closing ipset: %s", err)
	}

	s.Unlock()
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *FilteringConfig) {
	s.RLock()
	sc := s.conf.FilteringConfig
	*c = sc
	c.RatelimitWhitelist = aghstrings.CloneSlice(sc.RatelimitWhitelist)
	c.BootstrapDNS = aghstrings.CloneSlice(sc.BootstrapDNS)
	c.AllowedClients = aghstrings.CloneSlice(sc.AllowedClients)
	c.DisallowedClients = aghstrings.CloneSlice(sc.DisallowedClients)
	c.BlockedHosts = aghstrings.CloneSlice(sc.BlockedHosts)
	c.UpstreamDNS = aghstrings.CloneSlice(sc.UpstreamDNS)
	s.RUnlock()
}

// RDNSSettings returns the copy of actual RDNS configuration.
func (s *Server) RDNSSettings() (localPTRResolvers []string, resolveClients bool) {
	s.RLock()
	defer s.RUnlock()

	return aghstrings.CloneSlice(s.conf.LocalPTRResolvers), s.conf.ResolveClients
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

// RDNSExchanger is a resolver for clients' addresses.
type RDNSExchanger interface {
	// Exchange tries to resolve the ip in a suitable way, e.g. either as
	// local or as external.
	Exchange(ip net.IP) (host string, err error)
}

const (
	// rDNSEmptyAnswerErr is returned by Exchange method when the answer
	// section of respond is empty.
	rDNSEmptyAnswerErr agherr.Error = "the answer section is empty"

	// rDNSNotPTRErr is returned by Exchange method when the response is not
	// of PTR type.
	rDNSNotPTRErr agherr.Error = "the response is not a ptr"
)

// Exchange implements the RDNSExchanger interface for *Server.
func (s *Server) Exchange(ip net.IP) (host string, err error) {
	s.RLock()
	defer s.RUnlock()

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

	var resp *dns.Msg
	if s.subnetDetector.IsLocallyServedNetwork(ip) {
		err = s.localResolvers.Resolve(ctx)
	} else {
		err = s.internalProxy.Resolve(ctx)
	}
	if err != nil {
		return "", err
	}

	resp = ctx.Res

	if len(resp.Answer) == 0 {
		return "", fmt.Errorf("lookup for %q: %w", arpa, rDNSEmptyAnswerErr)
	}

	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		return "", fmt.Errorf("type checking: %w", rDNSNotPTRErr)
	}

	return strings.TrimSuffix(ptr.Ptr, "."), nil
}

// Start starts the DNS server.
func (s *Server) Start() error {
	s.Lock()
	defer s.Unlock()
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
// port numbers as a map.  For internal use only.
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

// setupResolvers initializes the resolvers for local addresses.  For internal
// use only.
func (s *Server) setupResolvers(localAddrs []string) (err error) {
	bootstraps := s.conf.BootstrapDNS
	if len(localAddrs) == 0 {
		var sysRes aghnet.SystemResolvers
		// TODO(e.burkov): Enable the refresher after the actual
		// implementation passes the public testing.
		sysRes, err = aghnet.NewSystemResolvers(0, nil)
		if err != nil {
			return err
		}

		localAddrs = sysRes.Get()
		bootstraps = nil
	}
	log.Debug("upstreams to resolve PTR for local addresses: %v", localAddrs)

	var ourAddrs []string
	ourAddrs, err = s.collectDNSIPAddrs()
	if err != nil {
		return err
	}

	ourAddrsSet := aghstrings.NewSet(ourAddrs...)

	// TODO(e.burkov): The approach of subtracting sets of strings is not
	// really applicable here since in case of listening on all network
	// interfaces we should check the whole interface's network to cut off
	// all the loopback addresses as well.
	localAddrs = aghstrings.FilterOut(localAddrs, ourAddrsSet.Has)

	var upsConfig proxy.UpstreamConfig
	upsConfig, err = proxy.ParseUpstreamsConfig(localAddrs, upstream.Options{
		Bootstrap: bootstraps,
		Timeout:   defaultLocalTimeout,
		// TODO(e.burkov): Should we verify server's ceritificates?
	})
	if err != nil {
		return fmt.Errorf("parsing upstreams: %w", err)
	}

	s.localResolvers = &proxy.Proxy{
		Config: proxy.Config{
			UpstreamConfig: &upsConfig,
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

	// Initialize IPSET configuration
	// --
	err := s.ipset.init(s.conf.IPSETList)
	if err != nil {
		if !errors.Is(err, os.ErrInvalid) && !errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("cannot initialize ipset: %w", err)
		}

		// ipset cannot currently be initialized if the server was
		// installed from Snap or when the user or the binary doesn't
		// have the required permissions, or when the kernel doesn't
		// support netfilter.
		//
		// Log and go on.
		//
		// TODO(a.garipov): The Snap problem can probably be solved if
		// we add the netlink-connector interface plug.
		log.Info("warning: cannot initialize ipset: %s", err)
	}

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

	return nil
}

// Stop stops the DNS server.
func (s *Server) Stop() error {
	s.Lock()
	defer s.Unlock()
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

// ServeHTTP is a HTTP handler method we use to provide DNS-over-HTTPS
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	p := s.dnsProxy
	s.RUnlock()
	if p != nil { // an attempt to protect against race in case we're here after Close() was called
		p.ServeHTTP(w, r)
	}
}

// IsBlockedIP - return TRUE if this client should be blocked
func (s *Server) IsBlockedIP(ip net.IP) (bool, string) {
	if ip == nil {
		return false, ""
	}

	return s.access.IsBlockedIP(ip)
}
