package dnsforward

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
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

var defaultDNS = []string{
	"https://dns10.quad9.net/dns-query",
}
var defaultBootstrap = []string{"9.9.9.10", "149.112.112.10", "2620:fe::10", "2620:fe::fe:10"}

// Often requested by all kinds of DNS probes
var defaultBlockedHosts = []string{"version.bind", "id.server", "hostname.bind"}

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
	dnsProxy   *proxy.Proxy          // DNS proxy instance
	dnsFilter  *dnsfilter.Dnsfilter  // DNS filter instance
	dhcpServer dhcpd.ServerInterface // DHCP server instance (optional)
	queryLog   querylog.QueryLog     // Query log instance
	stats      stats.Stats
	access     *accessCtx

	ipset ipsetCtx

	tableHostToIP     map[string]net.IP // "hostname -> IP" table for internal addresses (DHCP)
	tableHostToIPLock sync.Mutex

	tablePTR     map[string]string // "IP -> hostname" table for reverse lookup
	tablePTRLock sync.Mutex

	// DNS proxy instance for internal usage
	// We don't Start() it and so no listen port is required.
	internalProxy *proxy.Proxy

	isRunning bool

	sync.RWMutex
	conf ServerConfig
}

// DNSCreateParams - parameters for NewServer()
type DNSCreateParams struct {
	DNSFilter  *dnsfilter.Dnsfilter
	Stats      stats.Stats
	QueryLog   querylog.QueryLog
	DHCPServer dhcpd.ServerInterface
}

// NewServer creates a new instance of the dnsforward.Server
// Note: this function must be called only once
func NewServer(p DNSCreateParams) *Server {
	s := &Server{}
	s.dnsFilter = p.DNSFilter
	s.stats = p.Stats
	s.queryLog = p.QueryLog

	if p.DHCPServer != nil {
		s.dhcpServer = p.DHCPServer
		s.dhcpServer.SetOnLeaseChanged(s.onDHCPLeaseChanged)
		s.onDHCPLeaseChanged(dhcpd.LeaseChangedAdded)
	}

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
func (s *Server) Prepare(config *ServerConfig) error {
	// Initialize the server configuration
	// --
	if config != nil {
		s.conf = *config
		if s.conf.BlockingMode == "custom_ip" {
			s.conf.BlockingIPAddrv4 = net.ParseIP(s.conf.BlockingIPv4)
			s.conf.BlockingIPAddrv6 = net.ParseIP(s.conf.BlockingIPv6)
			if s.conf.BlockingIPAddrv4 == nil || s.conf.BlockingIPAddrv6 == nil {
				return fmt.Errorf("DNS: invalid custom blocking IP address specified")
			}
		}
		if s.conf.MaxGoroutines == 0 {
			s.conf.MaxGoroutines = 50
		}
	}

	// Set default values in the case if nothing is configured
	// --
	s.initDefaultSettings()

	// Initialize IPSET configuration
	// --
	s.ipset.init(s.conf.IPSETList)

	// Prepare DNS servers settings
	// --
	err := s.prepareUpstreamSettings()
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

	// Initialize DNS access module
	// --
	s.access = &accessCtx{}
	err = s.access.Init(s.conf.AllowedClients, s.conf.DisallowedClients, s.conf.BlockedHosts)
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

// IsBlockedIP - return TRUE if this client should be blocked
func (s *Server) IsBlockedIP(ip string) (bool, string) {
	return s.access.IsBlockedIP(ip)
}
