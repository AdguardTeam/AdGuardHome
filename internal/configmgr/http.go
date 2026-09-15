package configmgr

import (
	"fmt"
	"net/netip"
	"regexp"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// HTTPConfig is the on-disk web API configuration.
type HTTPConfig struct {
	// DoH contains DNS-over-HTTPS configuration.
	DoH *DoHConfig `yaml:"doh"`

	// Pprof defines the profiling HTTP handler.
	Pprof *HTTPPprofConfig `yaml:"pprof"`

	// Address is the addresses on which to serve web API.
	Address netip.AddrPort `yaml:"address"`

	// SessionTTL for a web session.
	SessionTTL timeutil.Duration `yaml:"session_ttl"`
}

// DoHConfig is the block with DNS-over-HTTPS configuration.
type DoHConfig struct {
	// Routes is the list of HTTP route patterns for DoH requests.  Each route
	// should be in the format "METHOD /path" or "METHOD /path/{param}".
	Routes []string `yaml:"routes"`

	// InsecureEnabled allows DoH queries via unencrypted HTTP.
	InsecureEnabled bool `yaml:"insecure_enabled"`
}

// HTTPPprofConfig is the block with pprof HTTP configuration.
type HTTPPprofConfig struct {
	// Port for the profiling handler.
	Port uint16 `yaml:"port"`

	// Enabled defines if the profiling handler is enabled.
	Enabled bool `yaml:"enabled"`
}

// type check
var _ validate.Interface = (*HTTPConfig)(nil)

// Validate implements the [validate.Interface] interface for *HTTPConfig.
func (c *HTTPConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	var errs []error
	errs = validate.Append(errs, "doh", c.DoH)
	errs = validate.Append(errs, "pprof", c.Pprof)

	return errors.Join(errs...)
}

// doHRoutePatternRegexp is a regular expression for validating DoH route
// patterns.
var doHRoutePatternRegexp = regexp.MustCompile(
	`^[A-Z]+ /[^\s{}]+(?:/[^\s{}]+)*(?:/\{[_A-Za-z][_A-Za-z0-9]*\})?$`,
)

// type check
var _ validate.Interface = (*DoHConfig)(nil)

// Validate implements the [validate.Interface] interface for *DoHConfig.
func (c *DoHConfig) Validate() (err error) {
	if c == nil {
		return nil
	}

	var errs []error
	for i, route := range c.Routes {
		if doHRoutePatternRegexp.MatchString(route) {
			continue
		}

		errs = append(errs, fmt.Errorf(`route %q at index %d: incorrect format`, route, i))
	}

	return errors.Join(errs...)
}

// type check
var _ validate.Interface = (*HTTPPprofConfig)(nil)

// Validate implements the [validate.Interface] interface for *HTTPPprofConfig.
func (c *HTTPPprofConfig) Validate() (err error) {
	// TODO(d.kolyshev):  Validate.

	return nil
}
