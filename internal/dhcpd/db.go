// On-disk database for lease table

package dhcpd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/renameio/v2/maybe"
)

const (
	// dataFilename contains saved leases.
	dataFilename = "leases.json"

	// dataVersion is the current version of the stored DHCP leases structure.
	dataVersion = 1
)

// dataLeases is the structure of the stored DHCP leases.
type dataLeases struct {
	// Version is the current version of the structure.
	Version int `json:"version"`

	// Leases is the list containing stored DHCP leases.
	Leases []*dbLease `json:"leases"`

	// V6Meta contains persisted IPv6 prefix-tracking metadata.
	V6Meta *dataLeasesV6Meta `json:"v6_meta,omitempty"`
}

// dataLeasesV6Meta is the persisted IPv6 prefix-tracking metadata.
type dataLeasesV6Meta struct {
	Deprecated []*dbDeprecatedPrefix `json:"deprecated_prefixes,omitempty"`
	Renewable  []netip.Prefix        `json:"renewable_prefixes,omitempty"`
}

// dbDeprecatedPrefix is one persisted deprecated IPv6 prefix and its expiry.
type dbDeprecatedPrefix struct {
	Prefix     netip.Prefix `json:"prefix"`
	ValidUntil string       `json:"valid_until"`
}

// v6MetaRestorer restores persisted DHCPv6 prefix metadata into the running
// server implementation.
type v6MetaRestorer interface {
	DHCPServer
	setRestoredPrefixMeta(
		renewable map[netip.Prefix]struct{},
		deprecated map[netip.Prefix]time.Time,
	)
}

// v6Snapshotter returns the DHCPv6 leases and prefix metadata for persistence.
type v6Snapshotter interface {
	DHCPServer
	dbSnapshot(now time.Time) (
		leases []*dhcpsvc.Lease,
		renewable map[netip.Prefix]struct{},
		deprecated map[netip.Prefix]time.Time,
	)
}

// dbLease is the structure of stored lease.
type dbLease struct {
	Expiry   string     `json:"expires"`
	IP       netip.Addr `json:"ip"`
	Hostname string     `json:"hostname"`
	HWAddr   string     `json:"mac"`
	IsStatic bool       `json:"static"`
}

// fromLease converts *dhcpsvc.Lease to *dbLease.
func fromLease(l *dhcpsvc.Lease) (dl *dbLease) {
	var expiryStr string
	if !l.IsStatic {
		// The front-end is waiting for RFC 3999 format of the time value.  It
		// also shouldn't got an Expiry field for static leases.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2692.
		expiryStr = l.Expiry.Format(time.RFC3339)
	}

	return &dbLease{
		Expiry:   expiryStr,
		Hostname: l.Hostname,
		HWAddr:   l.HWAddr.String(),
		IP:       l.IP,
		IsStatic: l.IsStatic,
	}
}

// toLease converts *dbLease to *dhcpsvc.Lease.
func (dl *dbLease) toLease() (l *dhcpsvc.Lease, err error) {
	mac, err := net.ParseMAC(dl.HWAddr)
	if err != nil {
		return nil, fmt.Errorf("parsing hardware address: %w", err)
	}

	expiry := time.Time{}
	if !dl.IsStatic {
		expiry, err = time.Parse(time.RFC3339, dl.Expiry)
		if err != nil {
			return nil, fmt.Errorf("parsing expiry time: %w", err)
		}
	}

	return &dhcpsvc.Lease{
		Expiry:   expiry,
		IP:       dl.IP,
		Hostname: dl.Hostname,
		HWAddr:   mac,
		IsStatic: dl.IsStatic,
	}, nil
}

// dbLoad loads stored leases.
func (s *server) dbLoad() (err error) {
	data, err := os.ReadFile(s.conf.dbFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading db: %w", err)
		}

		return nil
	}

	dl := &dataLeases{}
	err = json.Unmarshal(data, dl)
	if err != nil {
		return fmt.Errorf("decoding db: %w", err)
	}

	leases4, leases6 := splitStoredLeases(dl.Leases)

	err = s.srv4.ResetLeases(leases4)
	if err != nil {
		return fmt.Errorf("resetting dhcpv4 leases: %w", err)
	}

	if s.srv6 != nil {
		err = s.srv6.ResetLeases(leases6)
		if err != nil {
			return fmt.Errorf("resetting dhcpv6 leases: %w", err)
		}
		restoreLoadedV6Meta(s.srv6, dl.V6Meta)
	}

	log.Info(
		"dhcp: loaded leases v4:%d  v6:%d  total-read:%d from DB",
		len(leases4),
		len(leases6),
		len(dl.Leases),
	)

	return nil
}

// splitStoredLeases converts stored database leases into DHCPv4 and DHCPv6
// leases.
func splitStoredLeases(leases []*dbLease) (leases4, leases6 []*dhcpsvc.Lease) {
	for _, l := range leases {
		lease, err := l.toLease()
		if err != nil {
			log.Info("dhcp: invalid lease: %s", err)

			continue
		}

		if lease.IP.Is4() {
			leases4 = append(leases4, lease)
		} else {
			leases6 = append(leases6, lease)
		}
	}

	return leases4, leases6
}

// restoreLoadedV6Meta restores the persisted IPv6 prefix metadata into srv6.
func restoreLoadedV6Meta(srv6 DHCPServer, meta *dataLeasesV6Meta) {
	if meta == nil {
		return
	}

	v6srv, ok := srv6.(v6MetaRestorer)
	if !ok {
		return
	}

	renewable, deprecated := splitStoredV6Meta(meta)
	v6srv.setRestoredPrefixMeta(renewable, deprecated)
}

// splitStoredV6Meta converts stored IPv6 prefix metadata into the in-memory
// structures used by the DHCPv6 server.
func splitStoredV6Meta(meta *dataLeasesV6Meta) (
	renewable map[netip.Prefix]struct{},
	deprecated map[netip.Prefix]time.Time,
) {
	renewable = map[netip.Prefix]struct{}{}
	for _, pref := range meta.Renewable {
		renewable[pref] = struct{}{}
	}

	deprecated = map[netip.Prefix]time.Time{}
	for _, dp := range meta.Deprecated {
		if dp == nil {
			continue
		}

		until, err := time.Parse(time.RFC3339, dp.ValidUntil)
		if err != nil {
			log.Info("dhcp: invalid v6 deprecated prefix %s: %s", dp.Prefix, err)

			continue
		}

		deprecated[dp.Prefix] = until
	}

	return renewable, deprecated
}

// dbStore stores DHCP leases.
func (s *server) dbStore() (err error) {
	// Use an empty slice here as opposed to nil so that it doesn't write
	// "null" into the database file if leases are empty.
	leases := dbLeasesFromRef(s.srv4.getLeasesRef())
	var v6Meta *dataLeasesV6Meta

	if s.srv6 != nil {
		leases, v6Meta = s.dbStoreV6(leases)
	}

	return writeDB(s.conf.dbFilePath, leases, v6Meta)
}

// dbLeasesFromRef converts DHCP leases to database leases.
func dbLeasesFromRef(leases []*dhcpsvc.Lease) (dbLeases []*dbLease) {
	dbLeases = make([]*dbLease, 0, len(leases))
	for _, l := range leases {
		dbLeases = append(dbLeases, fromLease(l))
	}

	return dbLeases
}

// dbStoreV6 adds DHCPv6 leases and prefix metadata to the database snapshot.
func (s *server) dbStoreV6(leases []*dbLease) (out []*dbLease, v6Meta *dataLeasesV6Meta) {
	if srv6, ok := s.srv6.(v6Snapshotter); ok {
		leases6, renewable, deprecated := srv6.dbSnapshot(time.Now())
		leases = append(leases, dbLeasesFromRef(leases6)...)
		return leases, buildStoredV6Meta(renewable, deprecated)
	}

	return append(leases, dbLeasesFromRef(s.srv6.getLeasesRef())...), nil
}

// buildStoredV6Meta converts snapshot metadata into the persisted form.
func buildStoredV6Meta(
	renewable map[netip.Prefix]struct{},
	deprecated map[netip.Prefix]time.Time,
) (v6Meta *dataLeasesV6Meta) {
	if len(renewable) == 0 && len(deprecated) == 0 {
		return nil
	}

	v6Meta = &dataLeasesV6Meta{
		Renewable: make([]netip.Prefix, 0, len(renewable)),
	}
	for pref := range renewable {
		v6Meta.Renewable = append(v6Meta.Renewable, pref)
	}
	slices.SortFunc(v6Meta.Renewable, prefixCompare)

	if len(deprecated) == 0 {
		return v6Meta
	}

	v6Meta.Deprecated = make([]*dbDeprecatedPrefix, 0, len(deprecated))
	for pref, until := range deprecated {
		v6Meta.Deprecated = append(v6Meta.Deprecated, &dbDeprecatedPrefix{
			Prefix:     pref,
			ValidUntil: until.Format(time.RFC3339),
		})
	}
	slices.SortFunc(v6Meta.Deprecated, func(a, b *dbDeprecatedPrefix) int {
		return prefixCompare(a.Prefix, b.Prefix)
	})

	return v6Meta
}

// writeDB writes leases to file at path.
func writeDB(path string, leases []*dbLease, v6Meta *dataLeasesV6Meta) (err error) {
	defer func() { err = errors.Annotate(err, "writing db: %w") }()

	slices.SortFunc(leases, func(a, b *dbLease) (res int) {
		return strings.Compare(a.Hostname, b.Hostname)
	})

	dl := &dataLeases{
		Version: dataVersion,
		Leases:  leases,
		V6Meta:  v6Meta,
	}

	buf, err := json.Marshal(dl)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = maybe.WriteFile(path, buf, aghos.DefaultPermFile)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	log.Info("dhcp: stored %d leases in %q", len(leases), path)

	return nil
}
