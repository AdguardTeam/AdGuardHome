package dnsforward

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/ameshkov/dnscrypt/v2"
)

// BlockingMode is an enum of all allowed blocking modes.
type BlockingMode string

// Allowed blocking modes.
const (
	// BlockingModeCustomIP means respond with a custom IP address.
	BlockingModeCustomIP BlockingMode = "custom_ip"

	// BlockingModeDefault is the same as BlockingModeNullIP for
	// Adblock-style rules, but responds with the IP address specified in
	// the rule when blocked by an `/etc/hosts`-style rule.
	BlockingModeDefault BlockingMode = "default"

	// BlockingModeNullIP means respond with a zero IP address: "0.0.0.0"
	// for A requests and "::" for AAAA ones.
	BlockingModeNullIP BlockingMode = "null_ip"

	// BlockingModeNXDOMAIN means respond with the NXDOMAIN code.
	BlockingModeNXDOMAIN BlockingMode = "nxdomain"

	// BlockingModeREFUSED means respond with the REFUSED code.
	BlockingModeREFUSED BlockingMode = "refused"
)

// FilteringConfig represents the DNS filtering configuration of AdGuard Home
// The zero FilteringConfig is empty and ready for use.
type FilteringConfig struct {
	// Callbacks for other modules
	// --

	// FilterHandler is an optional additional filtering callback.
	FilterHandler func(clientAddr net.IP, clientID string, settings *filtering.Settings) `yaml:"-"`

	// GetCustomUpstreamByClient is a callback that returns upstreams
	// configuration based on the client IP address or ClientID.  It returns
	// nil if there are no custom upstreams for the client.
	GetCustomUpstreamByClient func(id string) (conf *proxy.UpstreamConfig, err error) `yaml:"-"`

	// Protection configuration
	// --

	ProtectionEnabled  bool         `yaml:"protection_enabled"`   // whether or not use any of filtering features
	BlockingMode       BlockingMode `yaml:"blocking_mode"`        // mode how to answer filtered requests
	BlockingIPv4       net.IP       `yaml:"blocking_ipv4"`        // IP address to be returned for a blocked A request
	BlockingIPv6       net.IP       `yaml:"blocking_ipv6"`        // IP address to be returned for a blocked AAAA request
	BlockedResponseTTL uint32       `yaml:"blocked_response_ttl"` // if 0, then default is used (3600)

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

	UpstreamDNS         []string `yaml:"upstream_dns"`
	UpstreamDNSFileName string   `yaml:"upstream_dns_file"`
	BootstrapDNS        []string `yaml:"bootstrap_dns"` // a list of bootstrap DNS for DoH and DoT (plain DNS only)
	AllServers          bool     `yaml:"all_servers"`   // if true, parallel queries to all configured upstream servers are enabled
	FastestAddr         bool     `yaml:"fastest_addr"`  // use Fastest Address algorithm

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
	// CacheOptimistic defines if optimistic cache mechanism should be used.
	CacheOptimistic bool `yaml:"cache_optimistic"`

	// Other settings
	// --

	BogusNXDomain          []string `yaml:"bogus_nxdomain"`     // transform responses with these IP addresses to NXDOMAIN
	AAAADisabled           bool     `yaml:"aaaa_disabled"`      // Respond with an empty answer to all AAAA requests
	EnableDNSSEC           bool     `yaml:"enable_dnssec"`      // Set DNSSEC flag in outcoming DNS request
	EnableEDNSClientSubnet bool     `yaml:"edns_client_subnet"` // Enable EDNS Client Subnet option
	MaxGoroutines          uint32   `yaml:"max_goroutines"`     // Max. number of parallel goroutines for processing incoming requests

	// IpsetList is the ipset configuration that allows AdGuard Home to add
	// IP addresses of the specified domain names to an ipset list.  Syntax:
	//
	//   DOMAIN[,DOMAIN].../IPSET_NAME
	//
	IpsetList []string `yaml:"ipset"`
}

// TLSConfig is the TLS configuration for HTTPS, DNS-over-HTTPS, and DNS-over-TLS
type TLSConfig struct {
	TLSListenAddrs  []*net.TCPAddr `yaml:"-" json:"-"`
	QUICListenAddrs []*net.UDPAddr `yaml:"-" json:"-"`

	// Reject connection if the client uses server name (in SNI) that doesn't match the certificate
	StrictSNICheck bool `yaml:"strict_sni_check" json:"-"`

	// PEM-encoded certificates chain
	CertificateChain string `yaml:"certificate_chain" json:"certificate_chain"`
	// PEM-encoded private key
	PrivateKey string `yaml:"private_key" json:"private_key"`

	CertificatePath string `yaml:"certificate_path" json:"certificate_path"`
	PrivateKeyPath  string `yaml:"private_key_path" json:"private_key_path"`

	CertificateChainData []byte `yaml:"-" json:"-"`
	PrivateKeyData       []byte `yaml:"-" json:"-"`

	// ServerName is the hostname of the server.  Currently, it is only
	// being used for client ID checking.
	ServerName string `yaml:"-" json:"-"`

	cert tls.Certificate
	// DNS names from certificate (SAN) or CN value from Subject
	dnsNames []string
}

// DNSCryptConfig is the DNSCrypt server configuration struct.
type DNSCryptConfig struct {
	UDPListenAddrs []*net.UDPAddr
	TCPListenAddrs []*net.TCPAddr
	ProviderName   string
	ResolverCert   *dnscrypt.Cert
	Enabled        bool
}

// ServerConfig represents server configuration.
// The zero ServerConfig is empty and ready for use.
type ServerConfig struct {
	UDPListenAddrs []*net.UDPAddr        // UDP listen address
	TCPListenAddrs []*net.TCPAddr        // TCP listen address
	UpstreamConfig *proxy.UpstreamConfig // Upstream DNS servers config
	OnDNSRequest   func(d *proxy.DNSContext)

	FilteringConfig
	TLSConfig
	DNSCryptConfig
	TLSAllowUnencryptedDoH bool

	// UpstreamTimeout is the timeout for querying upstream servers.
	UpstreamTimeout time.Duration

	TLSv12Roots *x509.CertPool // list of root CAs for TLSv1.2
	TLSCiphers  []uint16       // list of TLS ciphers to use

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))

	// ResolveClients signals if the RDNS should resolve clients' addresses.
	ResolveClients bool

	// UsePrivateRDNS defines if the PTR requests for unknown addresses from
	// locally-served networks should be resolved via private PTR resolvers.
	UsePrivateRDNS bool

	// LocalPTRResolvers is a slice of addresses to be used as upstreams for
	// resolving PTR queries for local addresses.
	LocalPTRResolvers []string
}

// if any of ServerConfig values are zero, then default values from below are used
var defaultValues = ServerConfig{
	UDPListenAddrs:  []*net.UDPAddr{{Port: 53}},
	TCPListenAddrs:  []*net.TCPAddr{{Port: 53}},
	FilteringConfig: FilteringConfig{BlockedResponseTTL: 3600},
}

// createProxyConfig creates and validates configuration for the main proxy
func (s *Server) createProxyConfig() (proxy.Config, error) {
	proxyConfig := proxy.Config{
		UDPListenAddr:          s.conf.UDPListenAddrs,
		TCPListenAddr:          s.conf.TCPListenAddrs,
		Ratelimit:              int(s.conf.Ratelimit),
		RatelimitWhitelist:     s.conf.RatelimitWhitelist,
		RefuseAny:              s.conf.RefuseAny,
		CacheMinTTL:            s.conf.CacheMinTTL,
		CacheMaxTTL:            s.conf.CacheMaxTTL,
		CacheOptimistic:        s.conf.CacheOptimistic,
		UpstreamConfig:         s.conf.UpstreamConfig,
		BeforeRequestHandler:   s.beforeRequestHandler,
		RequestHandler:         s.handleDNSRequest,
		EnableEDNSClientSubnet: s.conf.EnableEDNSClientSubnet,
		MaxGoroutines:          int(s.conf.MaxGoroutines),
	}

	if s.conf.CacheSize != 0 {
		proxyConfig.CacheEnabled = true
		proxyConfig.CacheSizeBytes = int(s.conf.CacheSize)
	}

	proxyConfig.UpstreamMode = proxy.UModeLoadBalance
	if s.conf.AllServers {
		proxyConfig.UpstreamMode = proxy.UModeParallel
	} else if s.conf.FastestAddr {
		proxyConfig.UpstreamMode = proxy.UModeFastestAddr
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

	if s.conf.DNSCryptConfig.Enabled {
		proxyConfig.DNSCryptUDPListenAddr = s.conf.DNSCryptConfig.UDPListenAddrs
		proxyConfig.DNSCryptTCPListenAddr = s.conf.DNSCryptConfig.TCPListenAddrs
		proxyConfig.DNSCryptProviderName = s.conf.DNSCryptConfig.ProviderName
		proxyConfig.DNSCryptResolverCert = s.conf.DNSCryptConfig.ResolverCert
	}

	// Validate proxy config
	if proxyConfig.UpstreamConfig == nil || len(proxyConfig.UpstreamConfig.Upstreams) == 0 {
		return proxyConfig, errors.Error("no default upstream servers configured")
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

	if s.conf.UDPListenAddrs == nil {
		s.conf.UDPListenAddrs = defaultValues.UDPListenAddrs
	}

	if s.conf.TCPListenAddrs == nil {
		s.conf.TCPListenAddrs = defaultValues.TCPListenAddrs
	}

	if len(s.conf.BlockedHosts) == 0 {
		s.conf.BlockedHosts = defaultBlockedHosts
	}

	if s.conf.UpstreamTimeout == 0 {
		s.conf.UpstreamTimeout = DefaultTimeout
	}
}

// prepareUpstreamSettings - prepares upstream DNS server settings
func (s *Server) prepareUpstreamSettings() error {
	// We're setting a customized set of RootCAs
	// The reason is that Go default mechanism of loading TLS roots
	// does not always work properly on some routers so we're
	// loading roots manually and pass it here.
	// See "util.LoadSystemRootCAs"
	upstream.RootCAs = s.conf.TLSv12Roots

	// See util.InitTLSCiphers -- removed unsafe ciphers
	if len(s.conf.TLSCiphers) > 0 {
		upstream.CipherSuites = s.conf.TLSCiphers
	}

	// Load upstreams either from the file, or from the settings
	var upstreams []string
	if s.conf.UpstreamDNSFileName != "" {
		data, err := os.ReadFile(s.conf.UpstreamDNSFileName)
		if err != nil {
			return err
		}
		d := string(data)
		for len(d) != 0 {
			s := aghstrings.SplitNext(&d, '\n')
			upstreams = append(upstreams, s)
		}
		log.Debug("dns: using %d upstream servers from file %s", len(upstreams), s.conf.UpstreamDNSFileName)
	} else {
		upstreams = s.conf.UpstreamDNS
	}

	upstreams = aghstrings.FilterOut(upstreams, aghstrings.IsCommentOrEmpty)
	upstreamConfig, err := proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{
			Bootstrap: s.conf.BootstrapDNS,
			Timeout:   s.conf.UpstreamTimeout,
		},
	)
	if err != nil {
		return fmt.Errorf("dns: proxy.ParseUpstreamsConfig: %w", err)
	}

	if len(upstreamConfig.Upstreams) == 0 {
		log.Info("warning: no default upstream servers specified, using %v", defaultDNS)
		var uc *proxy.UpstreamConfig
		uc, err = proxy.ParseUpstreamsConfig(
			defaultDNS,
			&upstream.Options{
				Bootstrap: s.conf.BootstrapDNS,
				Timeout:   s.conf.UpstreamTimeout,
			},
		)
		if err != nil {
			return fmt.Errorf("dns: failed to parse default upstreams: %v", err)
		}
		upstreamConfig.Upstreams = uc.Upstreams
	}

	s.conf.UpstreamConfig = upstreamConfig

	return nil
}

// prepareIntlProxy - initializes DNS proxy that we use for internal DNS queries
func (s *Server) prepareIntlProxy() {
	s.internalProxy = &proxy.Proxy{
		Config: proxy.Config{
			CacheEnabled:   true,
			CacheSizeBytes: 4096,
			UpstreamConfig: s.conf.UpstreamConfig,
		},
	}
}

// prepareTLS - prepares TLS configuration for the DNS proxy
func (s *Server) prepareTLS(proxyConfig *proxy.Config) error {
	if len(s.conf.CertificateChainData) == 0 || len(s.conf.PrivateKeyData) == 0 {
		return nil
	}

	if s.conf.TLSListenAddrs == nil && s.conf.QUICListenAddrs == nil {
		return nil
	}

	if s.conf.TLSListenAddrs != nil {
		proxyConfig.TLSListenAddr = s.conf.TLSListenAddrs
	}

	if s.conf.QUICListenAddrs != nil {
		proxyConfig.QUICListenAddr = s.conf.QUICListenAddrs
	}

	var err error
	s.conf.cert, err = tls.X509KeyPair(s.conf.CertificateChainData, s.conf.PrivateKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse TLS keypair: %w", err)
	}

	if s.conf.StrictSNICheck {
		var x *x509.Certificate
		x, err = x509.ParseCertificate(s.conf.cert.Certificate[0])
		if err != nil {
			return fmt.Errorf("x509.ParseCertificate(): %w", err)
		}
		if len(x.DNSNames) != 0 {
			s.conf.dnsNames = x.DNSNames
			log.Debug("dns: using DNS names from certificate's SAN: %v", x.DNSNames)
			sort.Strings(s.conf.dnsNames)
		} else {
			s.conf.dnsNames = append(s.conf.dnsNames, x.Subject.CommonName)
			log.Debug("dns: using DNS name from certificate's CN: %s", x.Subject.CommonName)
		}
	}

	proxyConfig.TLSConfig = &tls.Config{
		GetCertificate: s.onGetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	return nil
}

// isInSorted returns true if s is in the sorted slice strs.
func isInSorted(strs []string, s string) (ok bool) {
	i := sort.SearchStrings(strs, s)
	if i == len(strs) || strs[i] != s {
		return false
	}

	return true
}

// isWildcard returns true if host is a wildcard hostname.
func isWildcard(host string) (ok bool) {
	return len(host) >= 2 && host[0] == '*' && host[1] == '.'
}

// matchesDomainWildcard returns true if host matches the domain wildcard
// pattern pat.
func matchesDomainWildcard(host, pat string) (ok bool) {
	return isWildcard(pat) && strings.HasSuffix(host, pat[1:])
}

// anyNameMatches returns true if sni, the client's SNI value, matches any of
// the DNS names and patterns from certificate.  dnsNames must be sorted.
func anyNameMatches(dnsNames []string, sni string) (ok bool) {
	if aghnet.ValidateDomainName(sni) != nil {
		return false
	}

	if isInSorted(dnsNames, sni) {
		return true
	}

	for _, dn := range dnsNames {
		if matchesDomainWildcard(sni, dn) {
			return true
		}
	}

	return false
}

// Called by 'tls' package when Client Hello is received
// If the server name (from SNI) supplied by client is incorrect - we terminate the ongoing TLS handshake.
func (s *Server) onGetCertificate(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if s.conf.StrictSNICheck && !anyNameMatches(s.conf.dnsNames, ch.ServerName) {
		log.Info("dns: tls: unknown SNI in Client Hello: %s", ch.ServerName)
		return nil, fmt.Errorf("invalid SNI")
	}
	return &s.conf.cert, nil
}
