package client

import (
	"context"
	"encoding"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
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

// MustNewUID is a wrapper around [NewUID] that panics if there is an error.
func MustNewUID() (uid UID) {
	uid, err := NewUID()
	if err != nil {
		panic(fmt.Errorf("unexpected uuidv7 error: %w", err))
	}

	return uid
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

// Persistent contains information about persistent clients.
type Persistent struct {
	// UpstreamConfig is the custom upstream configuration for this client.  If
	// it's nil, it has not been initialized yet.  If it's non-nil and empty,
	// there are no valid upstreams.  If it's non-nil and non-empty, these
	// upstream must be used.
	UpstreamConfig *proxy.CustomUpstreamConfig

	// SafeSearch handles search engine hosts rewrites.
	SafeSearch filtering.SafeSearch

	// BlockedServices is the configuration of blocked services of a client.  It
	// must not be nil after initialization.
	BlockedServices *filtering.BlockedServices

	// Name of the persistent client.  Must not be empty.
	Name string

	// Tags is a list of client tags that categorize the client.
	Tags []string

	// Upstreams is a list of custom upstream DNS servers for the client.
	Upstreams []string

	// IPs is a list of IP addresses that identify the client.  The client must
	// have at least one ID (IP, subnet, MAC, or ClientID).
	IPs []netip.Addr

	// Subnets identifying the client.  The client must have at least one ID
	// (IP, subnet, MAC, or ClientID).
	//
	// TODO(s.chzhen):  Use netutil.Prefix.
	Subnets []netip.Prefix

	// MACs identifying the client.  The client must have at least one ID (IP,
	// subnet, MAC, or ClientID).
	MACs []net.HardwareAddr

	// ClientIDs identifying the client.  The client must have at least one ID
	// (IP, subnet, MAC, or ClientID).
	ClientIDs []string

	// UID is the unique identifier of the persistent client.
	UID UID

	// UpstreamsCacheSize is the cache size for custom upstreams.
	UpstreamsCacheSize uint32

	// UpstreamsCacheEnabled specifies whether custom upstreams are used.
	UpstreamsCacheEnabled bool

	// UseOwnSettings specifies whether custom filtering settings are used.
	UseOwnSettings bool

	// FilteringEnabled specifies whether filtering is enabled.
	FilteringEnabled bool

	// SafeBrowsingEnabled specifies whether safe browsing is enabled.
	SafeBrowsingEnabled bool

	// ParentalEnabled specifies whether parental control is enabled.
	ParentalEnabled bool

	// UseOwnBlockedServices specifies whether custom services are blocked.
	UseOwnBlockedServices bool

	// IgnoreQueryLog specifies whether the client requests are logged.
	IgnoreQueryLog bool

	// IgnoreStatistics  specifies whether the client requests are counted.
	IgnoreStatistics bool

	// SafeSearchConf is the safe search filtering configuration.
	//
	// TODO(d.kolyshev): Make SafeSearchConf a pointer.
	SafeSearchConf filtering.SafeSearchConfig
}

// validate returns an error if persistent client information contains errors.
// allTags must be sorted.
func (c *Persistent) validate(ctx context.Context, l *slog.Logger, allTags []string) (err error) {
	switch {
	case c.Name == "":
		return errors.Error("empty name")
	case c.IDsLen() == 0:
		return errors.Error("id required")
	case c.UID == UID{}:
		return errors.Error("uid required")
	}

	conf, err := proxy.ParseUpstreamsConfig(c.Upstreams, &upstream.Options{})
	if err != nil {
		return fmt.Errorf("invalid upstream servers: %w", err)
	}

	err = conf.Close()
	if err != nil {
		l.ErrorContext(ctx, "client: closing upstream config", slogutil.KeyError, err)
	}

	for _, t := range c.Tags {
		_, ok := slices.BinarySearch(allTags, t)
		if !ok {
			return fmt.Errorf("invalid tag: %q", t)
		}
	}

	// TODO(s.chzhen):  Move to the constructor.
	slices.Sort(c.Tags)

	return nil
}

// SetIDs parses a list of strings into typed fields and returns an error if
// there is one.
func (c *Persistent) SetIDs(ids []string) (err error) {
	for _, id := range ids {
		err = c.setID(id)
		if err != nil {
			return err
		}
	}

	slices.SortFunc(c.IPs, netip.Addr.Compare)

	// TODO(s.chzhen):  Use netip.PrefixCompare in Go 1.23.
	slices.SortFunc(c.Subnets, subnetCompare)
	slices.SortFunc(c.MACs, slices.Compare[net.HardwareAddr])
	slices.Sort(c.ClientIDs)

	return nil
}

// subnetCompare is a comparison function for the two subnets.  It returns -1 if
// x sorts before y, 1 if x sorts after y, and 0 if their relative sorting
// position is the same.
func subnetCompare(x, y netip.Prefix) (cmp int) {
	if x == y {
		return 0
	}

	xAddr, xBits := x.Addr(), x.Bits()
	yAddr, yBits := y.Addr(), y.Bits()
	if xBits == yBits {
		return xAddr.Compare(yAddr)
	}

	if xBits > yBits {
		return -1
	} else {
		return 1
	}
}

// setID parses id into typed field if there is no error.
func (c *Persistent) setID(id string) (err error) {
	if id == "" {
		return errors.Error("clientid is empty")
	}

	var ip netip.Addr
	if ip, err = netip.ParseAddr(id); err == nil {
		c.IPs = append(c.IPs, ip)

		return nil
	}

	var subnet netip.Prefix
	if subnet, err = netip.ParsePrefix(id); err == nil {
		c.Subnets = append(c.Subnets, subnet)

		return nil
	}

	var mac net.HardwareAddr
	if mac, err = net.ParseMAC(id); err == nil {
		c.MACs = append(c.MACs, mac)

		return nil
	}

	err = ValidateClientID(id)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	c.ClientIDs = append(c.ClientIDs, strings.ToLower(id))

	return nil
}

// ValidateClientID returns an error if id is not a valid ClientID.
//
// TODO(s.chzhen):  It's an exact copy of the [dnsforward.ValidateClientID] to
// avoid the import cycle.  Remove it.
func ValidateClientID(id string) (err error) {
	err = netutil.ValidateHostnameLabel(id)
	if err != nil {
		// Replace the domain name label wrapper with our own.
		return fmt.Errorf("invalid clientid %q: %w", id, errors.Unwrap(err))
	}

	return nil
}

// IDs returns a list of client IDs containing at least one element.
func (c *Persistent) IDs() (ids []string) {
	ids = make([]string, 0, c.IDsLen())

	for _, ip := range c.IPs {
		ids = append(ids, ip.String())
	}

	for _, subnet := range c.Subnets {
		ids = append(ids, subnet.String())
	}

	for _, mac := range c.MACs {
		ids = append(ids, mac.String())
	}

	return append(ids, c.ClientIDs...)
}

// IDsLen returns a length of client ids.
func (c *Persistent) IDsLen() (n int) {
	return len(c.IPs) + len(c.Subnets) + len(c.MACs) + len(c.ClientIDs)
}

// EqualIDs returns true if the ids of the current and previous clients are the
// same.
func (c *Persistent) EqualIDs(prev *Persistent) (equal bool) {
	return slices.Equal(c.IPs, prev.IPs) &&
		slices.Equal(c.Subnets, prev.Subnets) &&
		slices.EqualFunc(c.MACs, prev.MACs, slices.Equal[net.HardwareAddr]) &&
		slices.Equal(c.ClientIDs, prev.ClientIDs)
}

// ShallowClone returns a deep copy of the client, except upstreamConfig,
// safeSearchConf, SafeSearch fields, because it's difficult to copy them.
func (c *Persistent) ShallowClone() (clone *Persistent) {
	clone = &Persistent{}
	*clone = *c

	clone.BlockedServices = c.BlockedServices.Clone()
	clone.Tags = slices.Clone(c.Tags)
	clone.Upstreams = slices.Clone(c.Upstreams)

	clone.IPs = slices.Clone(c.IPs)
	clone.Subnets = slices.Clone(c.Subnets)
	clone.MACs = slices.Clone(c.MACs)
	clone.ClientIDs = slices.Clone(c.ClientIDs)

	return clone
}

// CloseUpstreams closes the client-specific upstream config of c if any.
func (c *Persistent) CloseUpstreams() (err error) {
	if c.UpstreamConfig != nil {
		if err = c.UpstreamConfig.Close(); err != nil {
			return fmt.Errorf("closing upstreams of client %q: %w", c.Name, err)
		}
	}

	return nil
}
