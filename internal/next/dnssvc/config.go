package dnssvc

import (
	"log/slog"
	"net/netip"
	"time"
)

// Config is the AdGuard Home DNS service configuration structure.
//
// TODO(a.garipov): Add timeout for incoming requests.
type Config struct {
	// Logger is used for logging the operation of the web API service.  It must
	// not be nil.
	Logger *slog.Logger

	// Addresses are the addresses on which to serve plain DNS queries.
	Addresses []netip.AddrPort

	// BootstrapServers are the addresses of DNS servers used for bootstrapping
	// the upstream DNS server addresses.
	BootstrapServers []string

	// UpstreamServers are the upstream DNS server addresses to use.
	UpstreamServers []string

	// DNS64Prefixes is a slice of NAT64 prefixes to be used for DNS64.  See
	// also [Config.UseDNS64].
	DNS64Prefixes []netip.Prefix

	// UpstreamTimeout is the timeout for upstream requests.
	UpstreamTimeout time.Duration

	// BootstrapPreferIPv6, if true, instructs the bootstrapper to prefer IPv6
	// addresses to IPv4 ones when bootstrapping.
	BootstrapPreferIPv6 bool

	// UseDNS64, if true, enables DNS64 protection for incoming requests.
	UseDNS64 bool
}
