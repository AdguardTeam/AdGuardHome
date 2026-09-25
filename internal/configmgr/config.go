package configmgr

import (
	"fmt"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// Config is the top-level on-disk configuration structure.
//
// TODO(d.kolyshev):  Use.
// TODO(d.kolyshev):  Add tests and contracts.
type Config struct {
	// DHCP is a block with DHCP configuration params.
	DHCP *DHCPConfig `yaml:"dhcp"`

	// DNSConfig is a block with DNS configuration params.
	DNSConfig *DNSConfig `yaml:"dns"`

	// HTTP is a block with web API configuration settings.
	HTTP *HTTPConfig `yaml:"http"`

	// Log is a block with log configuration settings.
	Log *LogConfig `yaml:"log"`

	// QueryLog is a block with query log configuration settings.
	QueryLog *QueryLogConfig `yaml:"querylog"`

	// Stats is a block with statistics configuration settings.
	Stats *StatsConfig `yaml:"statistics"`

	// TLS is a block with TLS configuration settings.
	TLS *TLSConfig `yaml:"tls"`

	// ProxyURL is the address of proxy server for the internal HTTP client.
	ProxyURL string `yaml:"http_proxy"`

	// Language is a two-letter ISO 639-1 language code.
	//
	// TODO(d.kolyshev):  Validate.
	Language string `yaml:"language"`

	// Theme is a web UI theme for current user.
	Theme string `yaml:"theme"`

	// Users are the clients capable for accessing the web interface.
	Users []*WebUser `yaml:"users"`

	// AuthAttempts is the maximum number of failed login attempts a user can do
	// before being blocked.
	AuthAttempts uint `yaml:"auth_attempts"`

	// AuthBlockMin is the duration in minutes, of the block of new login
	// attempts after AuthAttempts unsuccessful login attempts.
	AuthBlockMin uint `yaml:"block_auth_min"`

	// SchemaVersion is the version of the configuration schema.  See
	// [configmigrate.LastSchemaVersion].
	SchemaVersion uint `yaml:"schema_version"`

	// UnsafeUseCustomUpdateIndexURL is the URL to the custom update index.
	//
	// NOTE: It's only exists for testing purposes and should not be used in
	// release.
	UnsafeUseCustomUpdateIndexURL bool `yaml:"unsafe_use_custom_update_index_url,omitempty"`
}

// type check
var _ validate.Interface = (*Config)(nil)

// Validate implements the [validate.Interface] interface for *Config.
func (c *Config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// Keep this in the same order as the fields in the config.
	validators := container.KeyValues[string, validate.Interface]{{
		Key:   "dhcp",
		Value: c.DHCP,
	}, {
		Key:   "dns",
		Value: c.DNSConfig,
	}, {
		Key:   "http",
		Value: c.HTTP,
	}, {
		Key:   "log",
		Value: c.Log,
	}, {
		Key:   "querylog",
		Value: c.QueryLog,
	}, {
		Key:   "statistics",
		Value: c.Stats,
	}, {
		Key:   "tls",
		Value: c.TLS,
	}}

	var errs []error
	for _, kv := range validators {
		errs = validate.Append(errs, kv.Key, kv.Value)
	}

	if c.Theme != "" {
		_, err = NewTheme(c.Theme)
		if err != nil {
			errs = append(errs, fmt.Errorf("theme: %w", err))
		}
	}

	errs = validate.AppendSlice(errs, "users", c.Users)

	return errors.Join(errs...)
}
