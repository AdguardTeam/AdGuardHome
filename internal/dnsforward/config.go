package dnsforward

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/ameshkov/dnscrypt/v2"
	"golang.org/x/exp/slices"
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

	// FilterHandler is an optional additional filtering callback.
	FilterHandler func(clientAddr net.IP, clientID string, settings *filtering.Settings) `yaml:"-"`

	// GetCustomUpstreamByClient is a callback that returns upstreams
	// configuration based on the client IP address or ClientID.  It returns
	// nil if there are no custom upstreams for the client.
	GetCustomUpstreamByClient func(id string) (conf *proxy.UpstreamConfig, err error) `yaml:"-"`

	// Protection configuration

	// ProtectionEnabled defines whether or not use any of filtering features.
	ProtectionEnabled bool `yaml:"protection_enabled"`

	// BlockingMode defines the way how blocked responses are constructed.
	BlockingMode BlockingMode `yaml:"blocking_mode"`

	// BlockingIPv4 is the IP address to be returned for a blocked A request.
	BlockingIPv4 net.IP `yaml:"blocking_ipv4"`

	// BlockingIPv6 is the IP address to be returned for a blocked AAAA
	// request.
	BlockingIPv6 net.IP `yaml:"blocking_ipv6"`

	// BlockedResponseTTL is the time-to-live value for blocked responses.  If
	// 0, then default value is used (3600).
	BlockedResponseTTL uint32 `yaml:"blocked_response_ttl"`

	// ProtectionDisabledUntil is the timestamp until when the protection is
	// disabled.
	ProtectionDisabledUntil *time.Time `yaml:"protection_disabled_until"`

	// ParentalBlockHost is the IP (or domain name) which is used to respond to
	// DNS requests blocked by parental control.
	ParentalBlockHost string `yaml:"parental_block_host"`

	// SafeBrowsingBlockHost is the IP (or domain name) which is used to
	// respond to DNS requests blocked by safe-browsing.
	SafeBrowsingBlockHost string `yaml:"safebrowsing_block_host"`

	// Anti-DNS amplification

	// Ratelimit is the maximum number of requests per second from a given IP
	// (0 to disable).
	Ratelimit uint32 `yaml:"ratelimit"`

	// RatelimitWhitelist is the list of whitelisted client IP addresses.
	RatelimitWhitelist []string `yaml:"ratelimit_whitelist"`

	// RefuseAny, if true, refuse ANY requests.
	RefuseAny bool `yaml:"refuse_any"`

	// Upstream DNS servers configuration

	// UpstreamDNS is the list of upstream DNS servers.
	UpstreamDNS []string `yaml:"upstream_dns"`

	// UpstreamDNSFileName, if set, points to the file which contains upstream
	// DNS servers.
	UpstreamDNSFileName string `yaml:"upstream_dns_file"`

	// BootstrapDNS is the list of bootstrap DNS servers for DoH and DoT
	// resolvers (plain DNS only).
	BootstrapDNS []string `yaml:"bootstrap_dns"`

	// AllServers, if true, parallel queries to all configured upstream servers
	// are enabled.
	AllServers bool `yaml:"all_servers"`

	// FastestAddr, if true, use Fastest Address algorithm.
	FastestAddr bool `yaml:"fastest_addr"`

	// FastestTimeout replaces the default timeout for dialing IP addresses
	// when FastestAddr is true.
	FastestTimeout timeutil.Duration `yaml:"fastest_timeout"`

	// Access settings

	// AllowedClients is the slice of IP addresses, CIDR networks, and
	// ClientIDs of allowed clients.  If not empty, only these clients are
	// allowed, and [FilteringConfig.DisallowedClients] are ignored.
	AllowedClients []string `yaml:"allowed_clients"`

	// DisallowedClients is the slice of IP addresses, CIDR networks, and
	// ClientIDs of disallowed clients.
	DisallowedClients []string `yaml:"disallowed_clients"`

	// BlockedHosts is the list of hosts that should be blocked.
	BlockedHosts []string `yaml:"blocked_hosts"`

	// TrustedProxies is the list of IP addresses and CIDR networks to detect
	// proxy servers addresses the DoH requests from which should be handled.
	// The value of nil or an empty slice for this field makes Proxy not trust
	// any address.
	TrustedProxies []string `yaml:"trusted_proxies"`

	// DNS cache settings

	// CacheSize is the DNS cache size (in bytes).
	CacheSize uint32 `yaml:"cache_size"`

	// CacheMinTTL is the override TTL value (minimum) received from upstream
	// server.
	CacheMinTTL uint32 `yaml:"cache_ttl_min"`

	// CacheMaxTTL is the override TTL value (maximum) received from upstream
	// server.
	CacheMaxTTL uint32 `yaml:"cache_ttl_max"`

	// CacheOptimistic defines if optimistic cache mechanism should be used.
	CacheOptimistic bool `yaml:"cache_optimistic"`

	// Other settings

	// BogusNXDomain is the list of IP addresses, responses with them will be
	// transformed to NXDOMAIN.
	BogusNXDomain []string `yaml:"bogus_nxdomain"`

	// AAAADisabled, if true, respond with an empty answer to all AAAA
	// requests.
	AAAADisabled bool `yaml:"aaaa_disabled"`

	// EnableDNSSEC, if true, set AD flag in outcoming DNS request.
	EnableDNSSEC bool `yaml:"enable_dnssec"`

	// EDNSClientSubnet is the settings list for EDNS Client Subnet.
	EDNSClientSubnet *EDNSClientSubnet `yaml:"edns_client_subnet"`

	// MaxGoroutines is the max number of parallel goroutines for processing
	// incoming requests.
	MaxGoroutines uint32 `yaml:"max_goroutines"`

	// HandleDDR, if true, handle DDR requests
	HandleDDR bool `yaml:"handle_ddr"`

	// IpsetList is the ipset configuration that allows AdGuard Home to add IP
	// addresses of the specified domain names to an ipset list.  Syntax:
	//
	//	DOMAIN[,DOMAIN].../IPSET_NAME
	//
	// This field is ignored if [IpsetListFileName] is set.
	IpsetList []string `yaml:"ipset"`

	// IpsetListFileName, if set, points to the file with ipset configuration.
	// The format is the same as in [IpsetList].
	IpsetListFileName string `yaml:"ipset_file"`

	// BootstrapPreferIPv6, if true, instructs the bootstrapper to prefer IPv6
	// addresses to IPv4 ones for DoH, DoQ, and DoT.
	BootstrapPreferIPv6 bool `yaml:"bootstrap_prefer_ipv6"`
}

// EDNSClientSubnet is the settings list for EDNS Client Subnet.
type EDNSClientSubnet struct {
	// CustomIP for EDNS Client Subnet.
	CustomIP netip.Addr `yaml:"custom_ip"`

	// Enabled defines if EDNS Client Subnet is enabled.
	Enabled bool `yaml:"enabled"`

	// UseCustom defines if CustomIP should be used.
	UseCustom bool `yaml:"use_custom"`
}

// TLSConfig is the TLS configuration for HTTPS, DNS-over-HTTPS, and DNS-over-TLS
type TLSConfig struct {
	cert tls.Certificate

	TLSListenAddrs   []*net.TCPAddr `yaml:"-" json:"-"`
	QUICListenAddrs  []*net.UDPAddr `yaml:"-" json:"-"`
	HTTPSListenAddrs []*net.TCPAddr `yaml:"-" json:"-"`

	// PEM-encoded certificates chain
	CertificateChain string `yaml:"certificate_chain" json:"certificate_chain"`
	// PEM-encoded private key
	PrivateKey string `yaml:"private_key" json:"private_key"`

	CertificatePath string `yaml:"certificate_path" json:"certificate_path"`
	PrivateKeyPath  string `yaml:"private_key_path" json:"private_key_path"`

	CertificateChainData []byte `yaml:"-" json:"-"`
	PrivateKeyData       []byte `yaml:"-" json:"-"`

	// ServerName is the hostname of the server.  Currently, it is only being
	// used for ClientID checking and Discovery of Designated Resolvers (DDR).
	ServerName string `yaml:"-" json:"-"`

	// DNS names from certificate (SAN) or CN value from Subject
	dnsNames []string

	// OverrideTLSCiphers, when set, contains the names of the cipher suites to
	// use.  If the slice is empty, the default safe suites are used.
	OverrideTLSCiphers []string `yaml:"override_tls_ciphers,omitempty" json:"-"`

	// StrictSNICheck controls if the connections with SNI mismatching the
	// certificate's ones should be rejected.
	StrictSNICheck bool `yaml:"strict_sni_check" json:"-"`

	// hasIPAddrs is set during the certificate parsing and is true if the
	// configured certificate contains at least a single IP address.
	hasIPAddrs bool
}

// DNSCryptConfig is the DNSCrypt server configuration struct.
type DNSCryptConfig struct {
	ResolverCert   *dnscrypt.Cert
	ProviderName   string
	UDPListenAddrs []*net.UDPAddr
	TCPListenAddrs []*net.TCPAddr
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

	// TLSCiphers are the IDs of TLS cipher suites to use.
	TLSCiphers []uint16

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister aghhttp.RegisterFunc

	// LocalPTRResolvers is a slice of addresses to be used as upstreams for
	// resolving PTR queries for local addresses.
	LocalPTRResolvers []string

	// DNS64Prefixes is a slice of NAT64 prefixes to be used for DNS64.
	DNS64Prefixes []netip.Prefix

	// ResolveClients signals if the RDNS should resolve clients' addresses.
	ResolveClients bool

	// UsePrivateRDNS defines if the PTR requests for unknown addresses from
	// locally-served networks should be resolved via private PTR resolvers.
	UsePrivateRDNS bool

	// UseDNS64 defines if DNS64 is enabled for incoming requests.
	UseDNS64 bool

	// ServeHTTP3 defines if HTTP/3 is be allowed for incoming requests.
	ServeHTTP3 bool

	// UseHTTP3Upstreams defines if HTTP/3 is be allowed for DNS-over-HTTPS
	// upstreams.
	UseHTTP3Upstreams bool
}

// if any of ServerConfig values are zero, then default values from below are used
var defaultValues = ServerConfig{
	UDPListenAddrs:  []*net.UDPAddr{{Port: 53}},
	TCPListenAddrs:  []*net.TCPAddr{{Port: 53}},
	FilteringConfig: FilteringConfig{BlockedResponseTTL: 3600},
}

// createProxyConfig creates and validates configuration for the main proxy.
func (s *Server) createProxyConfig() (conf proxy.Config, err error) {
	srvConf := s.conf
	conf = proxy.Config{
		UDPListenAddr:          srvConf.UDPListenAddrs,
		TCPListenAddr:          srvConf.TCPListenAddrs,
		HTTP3:                  srvConf.ServeHTTP3,
		Ratelimit:              int(srvConf.Ratelimit),
		RatelimitWhitelist:     srvConf.RatelimitWhitelist,
		RefuseAny:              srvConf.RefuseAny,
		TrustedProxies:         srvConf.TrustedProxies,
		CacheMinTTL:            srvConf.CacheMinTTL,
		CacheMaxTTL:            srvConf.CacheMaxTTL,
		CacheOptimistic:        srvConf.CacheOptimistic,
		UpstreamConfig:         srvConf.UpstreamConfig,
		BeforeRequestHandler:   s.beforeRequestHandler,
		RequestHandler:         s.handleDNSRequest,
		EnableEDNSClientSubnet: srvConf.EDNSClientSubnet.Enabled,
		MaxGoroutines:          int(srvConf.MaxGoroutines),
		UseDNS64:               srvConf.UseDNS64,
		DNS64Prefs:             srvConf.DNS64Prefixes,
	}

	if srvConf.EDNSClientSubnet.UseCustom {
		// TODO(s.chzhen):  Use netip.Addr instead of net.IP inside dnsproxy.
		conf.EDNSAddr = net.IP(srvConf.EDNSClientSubnet.CustomIP.AsSlice())
	}

	if srvConf.CacheSize != 0 {
		conf.CacheEnabled = true
		conf.CacheSizeBytes = int(srvConf.CacheSize)
	}

	setProxyUpstreamMode(
		&conf,
		srvConf.AllServers,
		srvConf.FastestAddr,
		srvConf.FastestTimeout.Duration,
	)

	for i, s := range srvConf.BogusNXDomain {
		var subnet *net.IPNet
		subnet, err = netutil.ParseSubnet(s)
		if err != nil {
			log.Error("subnet at index %d: %s", i, err)

			continue
		}

		conf.BogusNXDomain = append(conf.BogusNXDomain, subnet)
	}

	err = s.prepareTLS(&conf)
	if err != nil {
		return proxy.Config{}, fmt.Errorf("validating tls: %w", err)
	}

	if c := srvConf.DNSCryptConfig; c.Enabled {
		conf.DNSCryptUDPListenAddr = c.UDPListenAddrs
		conf.DNSCryptTCPListenAddr = c.TCPListenAddrs
		conf.DNSCryptProviderName = c.ProviderName
		conf.DNSCryptResolverCert = c.ResolverCert
	}

	if conf.UpstreamConfig == nil || len(conf.UpstreamConfig.Upstreams) == 0 {
		return proxy.Config{}, errors.Error("no default upstream servers configured")
	}

	return conf, nil
}

const (
	defaultSafeBrowsingBlockHost = "standard-block.dns.adguard.com"
	defaultParentalBlockHost     = "family-block.dns.adguard.com"
)

// initDefaultSettings initializes default settings if nothing
// is configured
func (s *Server) initDefaultSettings() {
	if len(s.conf.UpstreamDNS) == 0 {
		s.conf.UpstreamDNS = defaultDNS
	}

	if len(s.conf.BootstrapDNS) == 0 {
		s.conf.BootstrapDNS = defaultBootstrap
	}

	if s.conf.ParentalBlockHost == "" {
		s.conf.ParentalBlockHost = defaultParentalBlockHost
	}

	if s.conf.SafeBrowsingBlockHost == "" {
		s.conf.SafeBrowsingBlockHost = defaultSafeBrowsingBlockHost
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

// UpstreamHTTPVersions returns the HTTP versions for upstream configuration
// depending on configuration.
func UpstreamHTTPVersions(http3 bool) (v []upstream.HTTPVersion) {
	if !http3 {
		return upstream.DefaultHTTPVersions
	}

	return []upstream.HTTPVersion{
		upstream.HTTPVersion3,
		upstream.HTTPVersion2,
		upstream.HTTPVersion11,
	}
}

// prepareUpstreamSettings - prepares upstream DNS server settings
func (s *Server) prepareUpstreamSettings() error {
	// We're setting a customized set of RootCAs.  The reason is that Go default
	// mechanism of loading TLS roots does not always work properly on some
	// routers so we're loading roots manually and pass it here.
	//
	// See [aghtls.SystemRootCAs].
	upstream.RootCAs = s.conf.TLSv12Roots
	upstream.CipherSuites = s.conf.TLSCiphers

	// Load upstreams either from the file, or from the settings
	var upstreams []string
	if s.conf.UpstreamDNSFileName != "" {
		data, err := os.ReadFile(s.conf.UpstreamDNSFileName)
		if err != nil {
			return fmt.Errorf("reading upstream from file: %w", err)
		}

		upstreams = stringutil.SplitTrimmed(string(data), "\n")

		log.Debug("dns: using %d upstream servers from file %s", len(upstreams), s.conf.UpstreamDNSFileName)
	} else {
		upstreams = s.conf.UpstreamDNS
	}

	httpVersions := UpstreamHTTPVersions(s.conf.UseHTTP3Upstreams)
	upstreams = stringutil.FilterOut(upstreams, IsCommentOrEmpty)
	upstreamConfig, err := proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{
			Bootstrap:    s.conf.BootstrapDNS,
			Timeout:      s.conf.UpstreamTimeout,
			HTTPVersions: httpVersions,
			PreferIPv6:   s.conf.BootstrapPreferIPv6,
		},
	)
	if err != nil {
		return fmt.Errorf("parsing upstream config: %w", err)
	}

	if len(upstreamConfig.Upstreams) == 0 {
		log.Info("warning: no default upstream servers specified, using %v", defaultDNS)
		var uc *proxy.UpstreamConfig
		uc, err = proxy.ParseUpstreamsConfig(
			defaultDNS,
			&upstream.Options{
				Bootstrap:    s.conf.BootstrapDNS,
				Timeout:      s.conf.UpstreamTimeout,
				HTTPVersions: httpVersions,
				PreferIPv6:   s.conf.BootstrapPreferIPv6,
			},
		)
		if err != nil {
			return fmt.Errorf("parsing default upstreams: %w", err)
		}

		upstreamConfig.Upstreams = uc.Upstreams
	}

	s.conf.UpstreamConfig = upstreamConfig

	return nil
}

// setProxyUpstreamMode sets the upstream mode and related settings in conf
// based on provided parameters.
func setProxyUpstreamMode(
	conf *proxy.Config,
	allServers bool,
	fastestAddr bool,
	fastestTimeout time.Duration,
) {
	if allServers {
		conf.UpstreamMode = proxy.UModeParallel
	} else if fastestAddr {
		conf.UpstreamMode = proxy.UModeFastestAddr
		conf.FastestPingTimeout = fastestTimeout
	} else {
		conf.UpstreamMode = proxy.UModeLoadBalance
	}
}

// prepareIpsetListSettings reads and prepares the ipset configuration either
// from a file or from the data in the configuration file.
func (s *Server) prepareIpsetListSettings() (err error) {
	fn := s.conf.IpsetListFileName
	if fn == "" {
		return s.ipset.init(s.conf.IpsetList)
	}

	// #nosec G304 -- Trust the path explicitly given by the user.
	data, err := os.ReadFile(fn)
	if err != nil {
		return err
	}

	ipsets := stringutil.SplitTrimmed(string(data), "\n")

	log.Debug("dns: using %d ipset rules from file %q", len(ipsets), fn)

	return s.ipset.init(ipsets)
}

// prepareTLS - prepares TLS configuration for the DNS proxy
func (s *Server) prepareTLS(proxyConfig *proxy.Config) (err error) {
	if len(s.conf.CertificateChainData) == 0 || len(s.conf.PrivateKeyData) == 0 {
		return nil
	}

	if s.conf.TLSListenAddrs == nil && s.conf.QUICListenAddrs == nil {
		return nil
	}

	proxyConfig.TLSListenAddr = aghalg.CoalesceSlice(
		s.conf.TLSListenAddrs,
		proxyConfig.TLSListenAddr,
	)

	proxyConfig.QUICListenAddr = aghalg.CoalesceSlice(
		s.conf.QUICListenAddrs,
		proxyConfig.QUICListenAddr,
	)

	s.conf.cert, err = tls.X509KeyPair(s.conf.CertificateChainData, s.conf.PrivateKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse TLS keypair: %w", err)
	}

	cert, err := x509.ParseCertificate(s.conf.cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("x509.ParseCertificate(): %w", err)
	}

	s.conf.hasIPAddrs = aghtls.CertificateHasIP(cert)

	if s.conf.StrictSNICheck {
		if len(cert.DNSNames) != 0 {
			s.conf.dnsNames = cert.DNSNames
			log.Debug("dns: using certificate's SAN as DNS names: %v", cert.DNSNames)
			slices.Sort(s.conf.dnsNames)
		} else {
			s.conf.dnsNames = append(s.conf.dnsNames, cert.Subject.CommonName)
			log.Debug("dns: using certificate's CN as DNS name: %s", cert.Subject.CommonName)
		}
	}

	proxyConfig.TLSConfig = &tls.Config{
		GetCertificate: s.onGetCertificate,
		CipherSuites:   s.conf.TLSCiphers,
		MinVersion:     tls.VersionTLS12,
	}

	return nil
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
	// Check sni is either a valid hostname or a valid IP address.
	if netutil.ValidateHostname(sni) != nil && net.ParseIP(sni) == nil {
		return false
	}

	if _, ok = slices.BinarySearch(dnsNames, sni); ok {
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

// UpdatedProtectionStatus updates protection state, if the protection was
// disabled temporarily.  Returns the updated state of protection.
func (s *Server) UpdatedProtectionStatus() (enabled bool, disabledUntil *time.Time) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	disabledUntil = s.conf.ProtectionDisabledUntil
	if disabledUntil == nil {
		return s.conf.ProtectionEnabled, nil
	}

	if time.Now().Before(*disabledUntil) {
		return false, disabledUntil
	}

	// Update the values in a separate goroutine, unless an update is already in
	// progress.  Since this method is called very often, and this update is a
	// relatively rare situation, do not lock s.serverLock for writing, as that
	// can lead to freezes.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/5661.
	if s.protectionUpdateInProgress.CompareAndSwap(false, true) {
		go s.enableProtectionAfterPause()
	}

	return true, nil
}

// enableProtectionAfterPause sets the protection configuration to enabled
// values.  It is intended to be used as a goroutine.
func (s *Server) enableProtectionAfterPause() {
	defer log.OnPanic("dns: enabling protection after pause")

	defer s.protectionUpdateInProgress.Store(false)

	defer s.conf.ConfigModified()

	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	s.conf.ProtectionEnabled = true
	s.conf.ProtectionDisabledUntil = nil

	log.Info("dns: protection is restarted after pause")
}
