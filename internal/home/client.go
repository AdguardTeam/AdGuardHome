package home

import (
	"encoding"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
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

	Tags      []string
	Upstreams []string

	IPs []netip.Addr
	// TODO(s.chzhen):  Use netutil.Prefix.
	Subnets   []netip.Prefix
	MACs      []net.HardwareAddr
	ClientIDs []string

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

// setTags sets the tags if they are known, otherwise logs an unknown tag.
func (c *persistentClient) setTags(tags []string, known *stringutil.Set) {
	for _, t := range tags {
		if !known.Has(t) {
			log.Info("skipping unknown tag %q", t)

			continue
		}

		c.Tags = append(c.Tags, t)
	}

	slices.Sort(c.Tags)
}

// setIDs parses a list of strings into typed fields and returns an error if
// there is one.
func (c *persistentClient) setIDs(ids []string) (err error) {
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
func (c *persistentClient) setID(id string) (err error) {
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

	err = dnsforward.ValidateClientID(id)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	c.ClientIDs = append(c.ClientIDs, strings.ToLower(id))

	return nil
}

// ids returns a list of client ids containing at least one element.
func (c *persistentClient) ids() (ids []string) {
	ids = make([]string, 0, c.idsLen())

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

// idsLen returns a length of client ids.
func (c *persistentClient) idsLen() (n int) {
	return len(c.IPs) + len(c.Subnets) + len(c.MACs) + len(c.ClientIDs)
}

// equalIDs returns true if the ids of the current and previous clients are the
// same.
func (c *persistentClient) equalIDs(prev *persistentClient) (equal bool) {
	return slices.Equal(c.IPs, prev.IPs) &&
		slices.Equal(c.Subnets, prev.Subnets) &&
		slices.EqualFunc(c.MACs, prev.MACs, slices.Equal[net.HardwareAddr]) &&
		slices.Equal(c.ClientIDs, prev.ClientIDs)
}

// shallowClone returns a deep copy of the client, except upstreamConfig,
// safeSearchConf, SafeSearch fields, because it's difficult to copy them.
func (c *persistentClient) shallowClone() (clone *persistentClient) {
	clone = &persistentClient{}
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
