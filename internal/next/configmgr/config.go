package configmgr

import (
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
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
var _ validator = (*config)(nil)

// validate implements the [validator] interface for *config.
func (c *config) validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(a.garipov): Add more validations.

	// Keep this in the same order as the fields in the config.
	validators := container.KeyValues[string, validator]{{
		Key:   "dns",
		Value: c.DNS,
	}, {
		Key:   "http",
		Value: c.HTTP,
	}, {
		Key:   "log",
		Value: c.Log,
	}}

	for _, kv := range validators {
		err = kv.Value.validate()
		if err != nil {
			return fmt.Errorf("%s: %w", kv.Key, err)
		}
	}

	return nil
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
var _ validator = (*dnsConfig)(nil)

// validate implements the [validator] interface for *dnsConfig.
//
// TODO(a.garipov): Add more validations.
func (c *dnsConfig) validate() (err error) {
	// TODO(a.garipov): Add more validations.
	switch {
	case c == nil:
		return errors.ErrNoValue
	case c.UpstreamTimeout.Duration <= 0:
		return newErrNotPositive("upstream_timeout", c.UpstreamTimeout)
	default:
		return nil
	}
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
var _ validator = (*httpConfig)(nil)

// validate implements the [validator] interface for *httpConfig.
//
// TODO(a.garipov): Add more validations.
func (c *httpConfig) validate() (err error) {
	switch {
	case c == nil:
		return errors.ErrNoValue
	case c.Timeout.Duration <= 0:
		return newErrNotPositive("timeout", c.Timeout)
	default:
		return c.Pprof.validate()
	}
}

// httpPprofConfig is the on-disk pprof configuration.
type httpPprofConfig struct {
	Port    uint16 `yaml:"port"`
	Enabled bool   `yaml:"enabled"`
}

// type check
var _ validator = (*httpPprofConfig)(nil)

// validate implements the [validator] interface for *httpPprofConfig.
func (c *httpPprofConfig) validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return nil
}

// logConfig is the on-disk web API configuration.
type logConfig struct {
	// TODO(a.garipov): Use.
	Verbose bool `yaml:"verbose"`
}

// type check
var _ validator = (*logConfig)(nil)

// validate implements the [validator] interface for *logConfig.
//
// TODO(a.garipov): Add more validations.
func (c *logConfig) validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return nil
}
