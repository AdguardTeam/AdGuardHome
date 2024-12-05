package websvc

import (
	"crypto/tls"
	"io/fs"
	"log/slog"
	"net/netip"
	"time"
)

// Config is the AdGuard Home web service configuration structure.
type Config struct {
	// Logger is used for logging the operation of the web API service.  It must
	// not be nil.
	Logger *slog.Logger

	// Pprof is the configuration for the pprof debug API.  It must not be nil.
	Pprof *PprofConfig

	// ConfigManager is used to show information about services as well as
	// dynamically reconfigure them.
	ConfigManager ConfigManager

	// Frontend is the filesystem with the frontend and other statically
	// compiled files.
	Frontend fs.FS

	// TLS is the optional TLS configuration.  If TLS is not nil,
	// SecureAddresses must not be empty.
	TLS *tls.Config

	// Start is the time of start of AdGuard Home.
	Start time.Time

	// OverrideAddress is the initial or override address for the HTTP API.  If
	// set, it is used instead of [Addresses] and [SecureAddresses].
	OverrideAddress netip.AddrPort

	// Addresses are the addresses on which to serve the plain HTTP API.
	Addresses []netip.AddrPort

	// SecureAddresses are the addresses on which to serve the HTTPS API.  If
	// SecureAddresses is not empty, TLS must not be nil.
	SecureAddresses []netip.AddrPort

	// Timeout is the timeout for all server operations.
	Timeout time.Duration

	// ForceHTTPS tells if all requests to Addresses should be redirected to a
	// secure address instead.
	//
	// TODO(a.garipov): Use; define rules, which address to redirect to.
	ForceHTTPS bool
}

// PprofConfig is the configuration for the pprof debug API.
type PprofConfig struct {
	Port    uint16 `yaml:"port"`
	Enabled bool   `yaml:"enabled"`
}

// Config returns the current configuration of the web service.  Config must not
// be called simultaneously with Start.  If svc was initialized with ":0"
// addresses, addrs will not return the actual bound ports until Start is
// finished.
func (svc *Service) Config() (c *Config) {
	c = &Config{
		Logger: svc.logger,
		Pprof: &PprofConfig{
			Port:    svc.pprofPort,
			Enabled: svc.pprof != nil,
		},
		ConfigManager: svc.confMgr,
		Frontend:      svc.frontend,
		TLS:           svc.tls,
		// Leave Addresses and SecureAddresses empty and get the actual
		// addresses that include the :0 ones later.
		Start:           svc.start,
		OverrideAddress: svc.overrideAddr,
		Timeout:         svc.timeout,
		ForceHTTPS:      svc.forceHTTPS,
	}

	c.Addresses, c.SecureAddresses = svc.addrs()

	return c
}
