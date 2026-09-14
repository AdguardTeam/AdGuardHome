package configmgr

import (
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// DNSConfig is a block with DNS configuration params.
type DNSConfig struct {
	// EDNSClientSubnet is the settings list for EDNS Client Subnet.
	EDNSClientSubnet *EDNSClientSubnet `yaml:"edns_client_subnet"`

	// PendingRequests configures duplicate requests policy.
	PendingRequests *PendingRequests `yaml:"pending_requests"`

	// IpsetListFileName, if set, points to the file with ipset configuration.
	// The format is the same as in [IpsetList].
	IpsetListFileName string `yaml:"ipset_file"`

	// UpstreamDNSFileName, if set, points to the file which contains upstream
	// DNS servers.
	UpstreamDNSFileName string `yaml:"upstream_dns_file"`

	// UpstreamMode determines the logic through which upstreams will be used.
	UpstreamMode string `yaml:"upstream_mode"`

	// BindHosts are the addresses to listen on.
	BindHosts []netip.Addr `yaml:"bind_hosts"`

	// RatelimitWhitelist is the list of whitelisted client IP addresses.
	RatelimitWhitelist []netip.Addr `yaml:"ratelimit_whitelist"`

	// DNS64Prefixes is the list of NAT64 prefixes to be used for DNS64.
	DNS64Prefixes []netip.Prefix `yaml:"dns64_prefixes"`

	// PrivateNets is the set of IP networks for which the private reverse DNS
	// resolver should be used.
	PrivateNets []netutil.Prefix `yaml:"private_networks"`

	// TrustedProxies is the list of CIDR networks with proxy servers addresses
	// from which the DoH requests should be handled.  The value of nil or an
	// empty slice for this field makes Proxy not trust any address.
	TrustedProxies []netutil.Prefix `yaml:"trusted_proxies"`

	// AllowedClients is the slice of IP addresses, CIDR networks, and ClientIDs
	// of allowed clients.  If not empty, only these clients are allowed, and
	// DisallowedClients are ignored.
	AllowedClients []string `yaml:"allowed_clients"`

	// BlockedHosts is the list of hosts that should be blocked.
	BlockedHosts []string `yaml:"blocked_hosts"`

	// BogusNXDomain is the list of IP addresses, responses with them will be
	// transformed to NXDOMAIN.
	BogusNXDomain []string `yaml:"bogus_nxdomain"`

	// BootstrapDNS is the list of bootstrap DNS servers for DoH and DoT
	// resolvers (plain DNS only).
	BootstrapDNS []string `yaml:"bootstrap_dns"`

	// DisallowedClients is the slice of IP addresses, CIDR networks, and
	// ClientIDs of disallowed clients.
	DisallowedClients []string `yaml:"disallowed_clients"`

	// FallbackDNS is the list of fallback DNS servers used when upstream DNS
	// servers are not responding.
	FallbackDNS []string `yaml:"fallback_dns"`

	// IpsetList is the ipset configuration that allows AdGuard Home to add IP
	// addresses of the specified domain names to an ipset list.  Syntax:
	//
	//	DOMAIN[,DOMAIN].../IPSET_NAME[,IPSET_NAME]...
	//
	// This field is ignored if [IpsetListFileName] is set.
	IpsetList []string `yaml:"ipset"`

	// PrivateRDNSResolvers is the slice of addresses to be used as upstreams
	// for private requests.  It's only used for PTR, SOA, and NS queries,
	// containing an ARPA subdomain, came from the the client with private
	// address.  The address considered private according to PrivateNets.
	//
	// If empty, the OS-provided resolvers are used for private requests.
	PrivateRDNSResolvers []string `yaml:"local_ptr_upstreams"`

	// UpstreamDNS is the list of upstream DNS servers.
	UpstreamDNS []string `yaml:"upstream_dns"`

	// CacheOptimisticAnswerTTL is the default TTL for expired cached responses.
	CacheOptimisticAnswerTTL timeutil.Duration `yaml:"cache_optimistic_answer_ttl"`

	// CacheOptimisticMaxAge is the maximum time entries remain in the cache
	// when cache is optimistic.
	CacheOptimisticMaxAge timeutil.Duration `yaml:"cache_optimistic_max_age"`

	// FastestTimeout replaces the default timeout for dialing IP addresses
	// when FastestAddr is true.
	FastestTimeout timeutil.Duration `yaml:"fastest_timeout"`

	// UpstreamTimeout is the timeout for querying upstream servers.
	UpstreamTimeout timeutil.Duration `yaml:"upstream_timeout"`

	// MaxGoroutines is the max number of parallel goroutines for processing
	// incoming requests.
	MaxGoroutines uint `yaml:"max_goroutines"`

	// RatelimitSubnetLenIPv4 is a subnet length for IPv4 addresses used for
	// rate limiting requests.
	RatelimitSubnetLenIPv4 uint `yaml:"ratelimit_subnet_len_ipv4"`

	// RatelimitSubnetLenIPv6 is a subnet length for IPv6 addresses used for
	// rate limiting requests.
	RatelimitSubnetLenIPv6 uint `yaml:"ratelimit_subnet_len_ipv6"`

	// CacheMaxTTL is the override TTL value (maximum) received from upstream
	// server.
	CacheMaxTTL uint32 `yaml:"cache_ttl_max"`

	// CacheMinTTL is the override TTL value (minimum) received from upstream
	// server.
	CacheMinTTL uint32 `yaml:"cache_ttl_min"`

	// CacheSize is the DNS cache size (in bytes).
	CacheSize uint32 `yaml:"cache_size"`

	// Ratelimit is the maximum number of requests per second from a given IP
	// (0 to disable).
	Ratelimit uint32 `yaml:"ratelimit"`

	// Port is the port to listen on.
	Port uint16 `yaml:"port"`

	// AAAADisabled, if true, respond with an empty answer to all AAAA
	// requests.
	AAAADisabled bool `yaml:"aaaa_disabled"`

	// AnonymizeClientIP defines if clients' IP addresses should be anonymized
	// in query log and statistics.
	AnonymizeClientIP bool `yaml:"anonymize_client_ip"`

	// BootstrapPreferIPv6, if true, instructs the bootstrapper to prefer IPv6
	// addresses to IPv4 ones for DoH, DoQ, and DoT.
	BootstrapPreferIPv6 bool `yaml:"bootstrap_prefer_ipv6"`

	// CacheEnabled defines if the DNS cache should be used.
	CacheEnabled bool `yaml:"cache_enabled"`

	// CacheOptimistic defines if optimistic cache mechanism should be used.
	CacheOptimistic bool `yaml:"cache_optimistic"`

	// EnableDNSSEC defines whether the proxy should set the AD/DO bits in the
	// upstream requests.
	EnableDNSSEC bool `yaml:"enable_dnssec"`

	// HandleDDR, if true, handle DDR requests
	HandleDDR bool `yaml:"handle_ddr"`

	// HostsFileEnabled defines whether to use information from the system hosts
	// file to resolve queries.
	HostsFileEnabled bool `yaml:"hostsfile_enabled"`

	// RefuseAny, if true, refuse ANY requests.
	RefuseAny bool `yaml:"refuse_any"`

	// ServeHTTP3 defines if HTTP/3 is allowed for incoming requests.
	//
	// TODO(a.garipov): Add to the UI when HTTP/3 support is no longer
	// experimental.
	ServeHTTP3 bool `yaml:"serve_http3"`

	// ServePlainDNS defines if plain DNS is allowed for incoming requests.
	ServePlainDNS bool `yaml:"serve_plain_dns"`

	// UseDNS64 defines if DNS64 should be used for incoming requests.  Requests
	// of type PTR for addresses within the configured prefixes will be resolved
	// via [PrivateRDNSResolvers], so those should be valid and UsePrivateRDNS
	// be set to true.
	UseDNS64 bool `yaml:"use_dns64"`

	// UseHTTP3Upstreams defines if HTTP/3 is allowed for DNS-over-HTTPS
	// upstreams.
	//
	// TODO(a.garipov): Add to the UI when HTTP/3 support is no longer
	// experimental.
	UseHTTP3Upstreams bool `yaml:"use_http3_upstreams"`

	// UsePrivateRDNS enables resolving requests containing a private IP address
	// using private reverse DNS resolvers.  See PrivateRDNSResolvers.
	//
	// TODO(e.burkov):  Rename in YAML.
	UsePrivateRDNS bool `yaml:"use_private_ptr_resolvers"`
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

// PendingRequests is a block with pending requests configuration.
type PendingRequests struct {
	// Enabled controls if duplicate requests should be sent to the upstreams
	// along with the original one.
	Enabled bool `yaml:"enabled"`
}

// type check
var _ validate.Interface = (*DNSConfig)(nil)

// Validate implements the [validate.Interface] interface for *DNSConfig.
func (c *DNSConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	var errs []error
	if c.UpstreamMode != "" {
		_, err = dnsforward.NewUpstreamMode(c.UpstreamMode)
		if err != nil {
			errs = append(errs, fmt.Errorf("upstream_mode: %w", err))
		}
	}

	// TODO(d.kolyshev):  Add more validations.

	return errors.Join(errs...)
}
