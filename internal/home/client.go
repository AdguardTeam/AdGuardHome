package home

import (
	"encoding"
	"fmt"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/google/uuid"
)

// UID is the type for the unique IDs of persistent clients.
type UID uuid.UUID

// NewUID returns a new persistent client UID.  Any error returned is an error
// from the cryptographic randomness reader.
func NewUID() (uid UID, err error) {
	uuidv7, err := uuid.NewV7()

	return UID(uuidv7), err
}

// type check
var _ encoding.TextMarshaler = UID{}

// MarshalText implements the [encoding.TextMarshaler] for UID.
func (uid UID) MarshalText() ([]byte, error) {
	return uuid.UUID(uid).MarshalText()
}

// type check
var _ encoding.TextUnmarshaler = (*UID)(nil)

// UnmarshalText implements the [encoding.TextUnmarshaler] interface for UID.
func (uid *UID) UnmarshalText(data []byte) error {
	return (*uuid.UUID)(uid).UnmarshalText(data)
}

// persistentClient contains information about persistent clients.
type persistentClient struct {
	// upstreamConfig is the custom upstream configuration for this client.  If
	// it's nil, it has not been initialized yet.  If it's non-nil and empty,
	// there are no valid upstreams.  If it's non-nil and non-empty, these
	// upstream must be used.
	upstreamConfig *proxy.CustomUpstreamConfig

	// TODO(d.kolyshev): Make safeSearchConf a pointer.
	safeSearchConf filtering.SafeSearchConfig
	SafeSearch     filtering.SafeSearch

	// BlockedServices is the configuration of blocked services of a client.
	BlockedServices *filtering.BlockedServices

	Name string

	IDs       []string
	Tags      []string
	Upstreams []string

	// UID is the unique identifier of the persistent client.
	UID UID

	UpstreamsCacheSize    uint32
	UpstreamsCacheEnabled bool

	UseOwnSettings        bool
	FilteringEnabled      bool
	SafeBrowsingEnabled   bool
	ParentalEnabled       bool
	UseOwnBlockedServices bool
	IgnoreQueryLog        bool
	IgnoreStatistics      bool
}

// ShallowClone returns a deep copy of the client, except upstreamConfig,
// safeSearchConf, SafeSearch fields, because it's difficult to copy them.
func (c *persistentClient) ShallowClone() (sh *persistentClient) {
	clone := *c

	clone.BlockedServices = c.BlockedServices.Clone()
	clone.IDs = stringutil.CloneSlice(c.IDs)
	clone.Tags = stringutil.CloneSlice(c.Tags)
	clone.Upstreams = stringutil.CloneSlice(c.Upstreams)

	return &clone
}

// closeUpstreams closes the client-specific upstream config of c if any.
func (c *persistentClient) closeUpstreams() (err error) {
	if c.upstreamConfig != nil {
		if err = c.upstreamConfig.Close(); err != nil {
			return fmt.Errorf("closing upstreams of client %q: %w", c.Name, err)
		}
	}

	return nil
}

// setSafeSearch initializes and sets the safe search filter for this client.
func (c *persistentClient) setSafeSearch(
	conf filtering.SafeSearchConfig,
	cacheSize uint,
	cacheTTL time.Duration,
) (err error) {
	ss, err := safesearch.NewDefault(conf, fmt.Sprintf("client %q", c.Name), cacheSize, cacheTTL)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	c.SafeSearch = ss

	return nil
}
