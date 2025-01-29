package configmgr

import (
	"net/netip"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// config is the top-level on-disk configuration structure.
type config struct {
	DNS  *dnsConfig  `yaml:"dns"`
	HTTP *httpConfig `yaml:"http"`
	Log  *logConfig  `yaml:"log"`
	// TODO(a.garipov): Use.
	SchemaVersion int `yaml:"schema_version"`
}

// type check
var _ validate.Interface = (*config)(nil)

// Validate implements the [validate.Interface] interface for *config.
func (c *config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(a.garipov): Add more validations.

	// Keep this in the same order as the fields in the config.
	validators := container.KeyValues[string, validate.Interface]{{
		Key:   "dns",
		Value: c.DNS,
	}, {
		Key:   "http",
		Value: c.HTTP,
	}, {
		Key:   "log",
		Value: c.Log,
	}}

	var errs []error
	for _, kv := range validators {
		errs = validate.Append(errs, kv.Key, kv.Value)
	}

	return errors.Join(errs...)
}

// dnsConfig is the on-disk DNS configuration.
type dnsConfig struct {
	Addresses           []netip.AddrPort  `yaml:"addresses"`
	BootstrapDNS        []string          `yaml:"bootstrap_dns"`
	UpstreamDNS         []string          `yaml:"upstream_dns"`
	DNS64Prefixes       []netip.Prefix    `yaml:"dns64_prefixes"`
	UpstreamTimeout     timeutil.Duration `yaml:"upstream_timeout"`
	BootstrapPreferIPv6 bool              `yaml:"bootstrap_prefer_ipv6"`
	UseDNS64            bool              `yaml:"use_dns64"`
}

// type check
var _ validate.Interface = (*dnsConfig)(nil)

// Validate implements the [validate.Interface] interface for *dnsConfig.
//
// TODO(a.garipov): Add more validations.
func (c *dnsConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(a.garipov): Add more validations.

	return validate.Positive("upstream_timeout", c.UpstreamTimeout)
}

// httpConfig is the on-disk web API configuration.
type httpConfig struct {
	Pprof *httpPprofConfig `yaml:"pprof"`

	// TODO(a.garipov): Document the configuration change.
	Addresses       []netip.AddrPort  `yaml:"addresses"`
	SecureAddresses []netip.AddrPort  `yaml:"secure_addresses"`
	Timeout         timeutil.Duration `yaml:"timeout"`
	ForceHTTPS      bool              `yaml:"force_https"`
}

// type check
var _ validate.Interface = (*httpConfig)(nil)

// Validate implements the [validate.Interface] interface for *httpConfig.
//
// TODO(a.garipov): Add more validations.
func (c *httpConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	errs := []error{
		validate.Positive("timeout", c.Timeout),
	}

	errs = validate.Append(errs, "pprof", c.Pprof)

	return errors.Join(errs...)
}

// httpPprofConfig is the on-disk pprof configuration.
type httpPprofConfig struct {
	Port    uint16 `yaml:"port"`
	Enabled bool   `yaml:"enabled"`
}

// type check
var _ validate.Interface = (*httpPprofConfig)(nil)

// Validate implements the [validate.Interface] interface for *httpPprofConfig.
func (c *httpPprofConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return nil
}

// logConfig is the on-disk web API configuration.
type logConfig struct {
	// TODO(a.garipov):  Use.
	Verbose bool `yaml:"verbose"`
}

// type check
var _ validate.Interface = (*logConfig)(nil)

// Validate implements the [validate.Interface] interface for *logConfig.
//
// TODO(a.garipov): Add more validations.
func (c *logConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return nil
}
