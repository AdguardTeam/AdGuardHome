package configmgr

import (
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/c2h5oh/datasize"
)

// FilteringConfig is the on-disk filtering configuration.
type FilteringConfig struct {
	// BlockingIPv4 is the IP address to be returned for a blocked A request.
	BlockingIPv4 netip.Addr `yaml:"blocking_ipv4"`

	// BlockingIPv6 is the IP address to be returned for a blocked AAAA request.
	BlockingIPv6 netip.Addr `yaml:"blocking_ipv6"`

	// BlockedServices is the configuration of blocked services.  Per-client
	// settings can override this configuration.
	BlockedServices *BlockedServices `yaml:"blocked_services"`

	// ProtectionDisabledUntil is the timestamp until when the protection is
	// disabled.
	ProtectionDisabledUntil *time.Time `yaml:"protection_disabled_until"`

	// SafeSearchConf is the safe search configuration.
	SafeSearchConf *SafeSearch `yaml:"safe_search"`

	// BlockingMode defines the way how blocked responses are constructed.
	BlockingMode string `yaml:"blocking_mode"`

	// ParentalBlockHost is the IP (or domain name) which is used to respond to
	// DNS requests blocked by parental control.
	ParentalBlockHost string `yaml:"parental_block_host"`

	// SafeBrowsingBlockHost is the IP (or domain name) which is used to respond
	// to DNS requests blocked by safe-browsing.
	SafeBrowsingBlockHost string `yaml:"safebrowsing_block_host"`

	// Rewrites is a list of legacy DNS rewrite records.
	Rewrites []*LegacyRewrite `yaml:"rewrites"`

	// SafeFSPatterns are the patterns for matching which local filtering-rule
	// files can be added.
	SafeFSPatterns []string `yaml:"safe_fs_patterns"`

	// MaxHTTPSize defines the maximum size of the HTTP body.  It must be
	// positive.
	MaxHTTPSize datasize.ByteSize `yaml:"max_http_size"`

	// ParentalCacheSize is the size of the parental control cache, in bytes.
	ParentalCacheSize uint `yaml:"parental_cache_size"`

	// SafeBrowsingCacheSize is the size of the safe browsing cache, in bytes.
	SafeBrowsingCacheSize uint `yaml:"safebrowsing_cache_size"`

	// SafeSearchCacheSize is the size of the safe search cache, in bytes.
	SafeSearchCacheSize uint `yaml:"safesearch_cache_size"`

	// CacheTime is the TTL of a cache element.
	CacheTime timeutil.Duration `yaml:"cache_time"`

	// BlockedResponseTTL is the time-to-live value for blocked responses.  If
	// 0, then default value is used (3600).
	BlockedResponseTTL uint32 `yaml:"blocked_response_ttl"`

	// FiltersUpdateIntervalHours is the time period to update filters
	// (in hours).
	FiltersUpdateIntervalHours uint32 `yaml:"filters_update_interval"`

	// FilteringEnabled indicates whether or not use filter lists.
	FilteringEnabled bool `yaml:"filtering_enabled"`

	// ParentalEnabled indicates whether parental control is enabled.
	ParentalEnabled bool `yaml:"parental_enabled"`

	// ProtectionEnabled defines whether or not use any of filtering features.
	ProtectionEnabled bool `yaml:"protection_enabled"`

	// RewritesEnabled indicates whether legacy rewrites are applied.
	RewritesEnabled bool `yaml:"rewrites_enabled"`

	// SafeBrowsingEnabled indicates whether safe browsing is enabled.
	SafeBrowsingEnabled bool `yaml:"safebrowsing_enabled"`
}

// BlockedServices is the configuration of blocked services.
type BlockedServices struct {
	// Schedule is blocked services schedule for every day of the week.
	Schedule *schedule.Weekly `yaml:"schedule"`

	// IDs is the names of blocked services.
	IDs []string `yaml:"ids"`
}

// SafeSearch is a struct with safe search related settings.
type SafeSearch struct {
	// Enabled indicates if safe search is enabled entirely.
	Enabled bool `yaml:"enabled"`

	// Services flags.  Each flag indicates if the corresponding service is
	// enabled or disabled.

	Bing       bool `yaml:"bing"`
	DuckDuckGo bool `yaml:"duckduckgo"`
	Ecosia     bool `yaml:"ecosia"`
	Google     bool `yaml:"google"`
	Pixabay    bool `yaml:"pixabay"`
	Yandex     bool `yaml:"yandex"`
	YouTube    bool `yaml:"youtube"`
}

// LegacyRewrite is a single legacy DNS rewrite record.
type LegacyRewrite struct {
	// Answer is the IP address, canonical name, or one of the special values:
	// "A" or "AAAA".
	Answer string `yaml:"answer"`

	// Domain is the pattern to which this rewrite applies.
	Domain string `yaml:"domain"`

	// Enabled indicates whether this rewrite is active.
	Enabled bool `yaml:"enabled"`
}

// type check
var _ validate.Interface = (*FilteringConfig)(nil)

// Validate implements the [validate.Interface] interface for *FilteringConfig.
func (c *FilteringConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(d.kolyshev):  Add more validations.

	return nil
}
