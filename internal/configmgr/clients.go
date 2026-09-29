package configmgr

import (
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/c2h5oh/datasize"
)

// ClientsConfig is the on-disk persistent client configuration.
type ClientsConfig struct {
	// Sources defines the set of sources to fetch the runtime clients from.
	Sources *ClientSourcesConfig `yaml:"runtime_sources"`

	// Persistent are the configured clients.
	Persistent []*Client `yaml:"persistent"`
}

// ClientSourcesConfig is used to configure where the runtime clients will be
// obtained from.
type ClientSourcesConfig struct {
	// ARP enables the ARP source of runtime clients.
	ARP bool `yaml:"arp"`

	// DHCP enables the DHCP source of runtime clients.
	DHCP bool `yaml:"dhcp"`

	// HostsFile enables the system hosts file source of runtime clients.
	HostsFile bool `yaml:"hosts"`

	// RDNS enables the reverse DNS source of runtime clients.
	RDNS bool `yaml:"rdns"`

	// WHOIS enables the WHOIS source of runtime clients.
	WHOIS bool `yaml:"whois"`
}

// Client represents the on-disk persistent client.
type Client struct {
	// BlockedServices is the configuration of blocked services of a client.
	BlockedServices *BlockedServices `yaml:"blocked_services"`

	// SafeSearchConf is the safe search configuration of a client.
	SafeSearchConf *SafeSearch `yaml:"safe_search"`

	// Name is the human-readable name of the client.
	Name string `yaml:"name"`

	// IDs are the identifiers of the client, such as IP addresses, CIDRs, MAC
	// addresses, or ClientIDs.
	IDs []string `yaml:"ids"`

	// Tags are the tags of the client.
	Tags []string `yaml:"tags"`

	// Upstreams are the custom upstream DNS servers of the client.
	Upstreams []string `yaml:"upstreams"`

	// UID is the unique identifier of the persistent client.
	UID client.UID `yaml:"uid"`

	// UpstreamsCacheSize is the DNS cache size.
	UpstreamsCacheSize datasize.ByteSize `yaml:"upstreams_cache_size"`

	// FilteringEnabled indicates if filtering is enabled for the client.
	FilteringEnabled bool `yaml:"filtering_enabled"`

	// IgnoreQueryLog indicates if the client's queries are excluded from the
	// query log.
	IgnoreQueryLog bool `yaml:"ignore_querylog"`

	// IgnoreStatistics indicates if the client's queries are excluded from the
	// statistics.
	IgnoreStatistics bool `yaml:"ignore_statistics"`

	// ParentalEnabled indicates if parental control is enabled for the client.
	ParentalEnabled bool `yaml:"parental_enabled"`

	// SafeBrowsingEnabled indicates if safe browsing is enabled for the client.
	SafeBrowsingEnabled bool `yaml:"safebrowsing_enabled"`

	// UpstreamsCacheEnabled indicates if the DNS cache is enabled.
	UpstreamsCacheEnabled bool `yaml:"upstreams_cache_enabled"`

	// UseGlobalBlockedServices indicates if the client uses the global blocked
	// services configuration.
	UseGlobalBlockedServices bool `yaml:"use_global_blocked_services"`

	// UseGlobalSettings indicates if the client uses the global filtering
	// settings.
	UseGlobalSettings bool `yaml:"use_global_settings"`
}

// type check
var _ validate.Interface = (*ClientsConfig)(nil)

// Validate implements the [validate.Interface] interface for *ClientsConfig.
func (c *ClientsConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return errors.Join(
		validate.NotNil("runtime_sources", c.Sources),
		validate.Slice("persistent", c.Persistent),
	)
}

// type check
var _ validate.Interface = (*Client)(nil)

// Validate implements the [validate.Interface] interface for *Client.
func (c *Client) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(d.kolyshev):  Add more validations.

	return nil
}
