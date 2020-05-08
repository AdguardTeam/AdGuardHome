package dnsforward

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"

	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
)

// FilteringConfig represents the DNS filtering configuration of AdGuard Home
// The zero FilteringConfig is empty and ready for use.
type FilteringConfig struct {
	// Callbacks for other modules
	// --

	// Filtering callback function
	FilterHandler func(clientAddr string, settings *dnsfilter.RequestFilteringSettings) `yaml:"-"`

	// This callback function returns the list of upstream servers for a client specified by IP address
	GetUpstreamsByClient func(clientAddr string) []upstream.Upstream `yaml:"-"`

	// Protection configuration
	// --

	ProtectionEnabled  bool   `yaml:"protection_enabled"` // whether or not use any of dnsfilter features
	BlockingMode       string `yaml:"blocking_mode"`      // mode how to answer filtered requests
	BlockingIPv4       string `yaml:"blocking_ipv4"`      // IP address to be returned for a blocked A request
	BlockingIPv6       string `yaml:"blocking_ipv6"`      // IP address to be returned for a blocked AAAA request
	BlockingIPAddrv4   net.IP `yaml:"-"`
	BlockingIPAddrv6   net.IP `yaml:"-"`
	BlockedResponseTTL uint32 `yaml:"blocked_response_ttl"` // if 0, then default is used (3600)

	// IP (or domain name) which is used to respond to DNS requests blocked by parental control or safe-browsing
	ParentalBlockHost     string `yaml:"parental_block_host"`
	SafeBrowsingBlockHost string `yaml:"safebrowsing_block_host"`

	// Anti-DNS amplification
	// --

	Ratelimit          uint32   `yaml:"ratelimit"`           // max number of requests per second from a given IP (0 to disable)
	RatelimitWhitelist []string `yaml:"ratelimit_whitelist"` // a list of whitelisted client IP addresses
	RefuseAny          bool     `yaml:"refuse_any"`          // if true, refuse ANY requests

	// Upstream DNS servers configuration
	// --

	UpstreamDNS  []string `yaml:"upstream_dns"`
	BootstrapDNS []string `yaml:"bootstrap_dns"` // a list of bootstrap DNS for DoH and DoT (plain DNS only)
	AllServers   bool     `yaml:"all_servers"`   // if true, parallel queries to all configured upstream servers are enabled
	FastestAddr  bool     `yaml:"fastest_addr"`  // use Fastest Address algorithm

	// Access settings
	// --

	AllowedClients    []string `yaml:"allowed_clients"`    // IP addresses of whitelist clients
	DisallowedClients []string `yaml:"disallowed_clients"` // IP addresses of clients that should be blocked
	BlockedHosts      []string `yaml:"blocked_hosts"`      // hosts that should be blocked

	// DNS cache settings
	// --

	CacheSize   uint32 `yaml:"cache_size"`    // DNS cache size (in bytes)
	CacheMinTTL uint32 `yaml:"cache_ttl_min"` // override TTL value (minimum) received from upstream server
	CacheMaxTTL uint32 `yaml:"cache_ttl_max"` // override TTL value (maximum) received from upstream server

	// Other settings
	// --

	BogusNXDomain          []string `yaml:"bogus_nxdomain"`     // transform responses with these IP addresses to NXDOMAIN
	AAAADisabled           bool     `yaml:"aaaa_disabled"`      // Respond with an empty answer to all AAAA requests
	EnableDNSSEC           bool     `yaml:"enable_dnssec"`      // Set DNSSEC flag in outcoming DNS request
	EnableEDNSClientSubnet bool     `yaml:"edns_client_subnet"` // Enable EDNS Client Subnet option
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
	TLSAllowUnencryptedDOH bool

	TLSv12Roots *x509.CertPool // list of root CAs for TLSv1.2
	TLSCiphers  []uint16       // list of TLS ciphers to use

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

// createProxyConfig creates and validates configuration for the main proxy
func (s *Server) createProxyConfig() (proxy.Config, error) {
	proxyConfig := proxy.Config{
		UDPListenAddr:            s.conf.UDPListenAddr,
		TCPListenAddr:            s.conf.TCPListenAddr,
		Ratelimit:                int(s.conf.Ratelimit),
		RatelimitWhitelist:       s.conf.RatelimitWhitelist,
		RefuseAny:                s.conf.RefuseAny,
		CacheEnabled:             true,
		CacheSizeBytes:           int(s.conf.CacheSize),
		CacheMinTTL:              s.conf.CacheMinTTL,
		CacheMaxTTL:              s.conf.CacheMaxTTL,
		Upstreams:                s.conf.Upstreams,
		DomainsReservedUpstreams: s.conf.DomainsReservedUpstreams,
		BeforeRequestHandler:     s.beforeRequestHandler,
		RequestHandler:           s.handleDNSRequest,
		AllServers:               s.conf.AllServers,
		EnableEDNSClientSubnet:   s.conf.EnableEDNSClientSubnet,
		FindFastestAddr:          s.conf.FastestAddr,
	}

	if len(s.conf.BogusNXDomain) > 0 {
		for _, s := range s.conf.BogusNXDomain {
			ip := net.ParseIP(s)
			if ip == nil {
				log.Error("Invalid bogus IP: %s", s)
			} else {
				proxyConfig.BogusNXDomain = append(proxyConfig.BogusNXDomain, ip)
			}
		}
	}

	// TLS settings
	err := s.prepareTLS(&proxyConfig)
	if err != nil {
		return proxyConfig, err
	}

	// Validate proxy config
	if len(proxyConfig.Upstreams) == 0 {
		return proxyConfig, errors.New("no upstream servers configured")
	}

	return proxyConfig, nil
}

// initDefaultSettings initializes default settings if nothing
// is configured
func (s *Server) initDefaultSettings() {
	if len(s.conf.UpstreamDNS) == 0 {
		s.conf.UpstreamDNS = defaultDNS
	}
	if len(s.conf.BootstrapDNS) == 0 {
		s.conf.BootstrapDNS = defaultBootstrap
	}
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
}

// prepareUpstreamSettings - prepares upstream DNS server settings
func (s *Server) prepareUpstreamSettings() error {
	upstreamConfig, err := proxy.ParseUpstreamsConfig(s.conf.UpstreamDNS, s.conf.BootstrapDNS, DefaultTimeout)
	if err != nil {
		return fmt.Errorf("DNS: proxy.ParseUpstreamsConfig: %s", err)
	}
	s.conf.Upstreams = upstreamConfig.Upstreams
	s.conf.DomainsReservedUpstreams = upstreamConfig.DomainReservedUpstreams
	return nil
}

// prepareIntlProxy - initializes DNS proxy that we use for internal DNS queries
func (s *Server) prepareIntlProxy() {
	intlProxyConfig := proxy.Config{
		CacheEnabled:             true,
		CacheSizeBytes:           4096,
		Upstreams:                s.conf.Upstreams,
		DomainsReservedUpstreams: s.conf.DomainsReservedUpstreams,
	}
	s.internalProxy = &proxy.Proxy{Config: intlProxyConfig}
}

// prepareTLS - prepares TLS configuration for the DNS proxy
func (s *Server) prepareTLS(proxyConfig *proxy.Config) error {
	if s.conf.TLSListenAddr != nil && len(s.conf.CertificateChainData) != 0 && len(s.conf.PrivateKeyData) != 0 {
		proxyConfig.TLSListenAddr = s.conf.TLSListenAddr
		var err error
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
	upstream.RootCAs = s.conf.TLSv12Roots
	upstream.CipherSuites = s.conf.TLSCiphers
	return nil
}

// Called by 'tls' package when Client Hello is received
// If the server name (from SNI) supplied by client is incorrect - we terminate the ongoing TLS handshake.
func (s *Server) onGetCertificate(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if s.conf.StrictSNICheck && !matchDNSName(s.conf.dnsNames, ch.ServerName) {
		log.Info("DNS: TLS: unknown SNI in Client Hello: %s", ch.ServerName)
		return nil, fmt.Errorf("invalid SNI")
	}
	return &s.conf.cert, nil
}
