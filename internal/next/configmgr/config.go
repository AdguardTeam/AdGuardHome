package configmgr

import (
	"net/netip"

	"github.com/AdguardTeam/golibs/timeutil"
)

// Configuration Structures

// config is the top-level on-disk configuration structure.
type config struct {
	DNS  *dnsConfig  `yaml:"dns"`
	HTTP *httpConfig `yaml:"http"`
	// TODO(a.garipov): Use.
	SchemaVersion int `yaml:"schema_version"`
	// TODO(a.garipov): Use.
	DebugPprof bool `yaml:"debug_pprof"`
	Verbose    bool `yaml:"verbose"`
}

// dnsConfig is the on-disk DNS configuration.
//
// TODO(a.garipov): Validate.
type dnsConfig struct {
	Addresses           []netip.AddrPort  `yaml:"addresses"`
	BootstrapDNS        []string          `yaml:"bootstrap_dns"`
	UpstreamDNS         []string          `yaml:"upstream_dns"`
	DNS64Prefixes       []netip.Prefix    `yaml:"dns64_prefixes"`
	UpstreamTimeout     timeutil.Duration `yaml:"upstream_timeout"`
	BootstrapPreferIPv6 bool              `yaml:"bootstrap_prefer_ipv6"`
	UseDNS64            bool              `yaml:"use_dns64"`
}

// httpConfig is the on-disk web API configuration.
//
// TODO(a.garipov): Validate.
type httpConfig struct {
	Addresses       []netip.AddrPort  `yaml:"addresses"`
	SecureAddresses []netip.AddrPort  `yaml:"secure_addresses"`
	Timeout         timeutil.Duration `yaml:"timeout"`
	ForceHTTPS      bool              `yaml:"force_https"`
}
