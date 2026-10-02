package configmgr

import (
	"fmt"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// HTTPConfig is the on-disk web API configuration.
type HTTPConfig struct {
	// DoH contains DNS-over-HTTPS configuration.  It must not be nil.
	DoH *DoHConfig `yaml:"doh"`

	// Pprof defines the profiling HTTP handler.  It must not be nil.
	Pprof *HTTPPprofConfig `yaml:"pprof"`

	// Address is the addresses on which to serve web API.
	Address netip.AddrPort `yaml:"address"`

	// SessionTTL for a web session.
	SessionTTL timeutil.Duration `yaml:"session_ttl"`
}

// DoHConfig is the block with DNS-over-HTTPS configuration.
type DoHConfig struct {
	// Routes is the list of HTTP route patterns for DoH requests.  Each route
	// must be a valid [http.ServeMux] pattern, and the routes must not conflict
	// with each other.
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

// type check
var _ validate.Interface = (*DoHConfig)(nil)

// Validate implements the [validate.Interface] interface for *DoHConfig.
func (c *DoHConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	if err = validateRoutes(c.Routes); err != nil {
		return fmt.Errorf("routes: %w", err)
	}

	return nil
}

// validateRoutes registers the given routes on a temporary [http.ServeMux] and
// returns an error if any of them cannot be registered.
func validateRoutes(routes []string) (err error) {
	mux := http.NewServeMux()

	var errs []error
	for i, route := range routes {
		if err = validateRoute(mux, route); err != nil {
			errs = append(errs, fmt.Errorf("route %q at index %d: %w", route, i, err))
		}
	}

	return errors.Join(errs...)
}

// validateRoute tries registering route on mux, converting the panic that
// [http.ServeMux.Handle] raises for an invalid or conflicting pattern into an
// error.  mux must not be nil.
func validateRoute(mux *http.ServeMux, route string) (err error) {
	defer func() {
		err = errors.FromRecovered(recover())
	}()

	mux.Handle(route, http.NotFoundHandler())

	return nil
}

// type check
var _ validate.Interface = (*HTTPPprofConfig)(nil)

// Validate implements the [validate.Interface] interface for *HTTPPprofConfig.
func (c *HTTPPprofConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(d.kolyshev):  Validate.

	return nil
}
