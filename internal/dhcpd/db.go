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

	leases := dl.Leases
	leases4 := []*dhcpsvc.Lease{}
	leases6 := []*dhcpsvc.Lease{}

	for _, l := range leases {
		var lease *dhcpsvc.Lease
		lease, err = l.toLease()
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

	err = s.srv4.ResetLeases(leases4)
	if err != nil {
		return fmt.Errorf("resetting dhcpv4 leases: %w", err)
	}

	if s.srv6 != nil {
		err = s.srv6.ResetLeases(leases6)
		if err != nil {
			return fmt.Errorf("resetting dhcpv6 leases: %w", err)
		}
		if dl.V6Meta != nil {
			if srv6, ok := s.srv6.(*v6Server); ok {
				renewable := map[netip.Prefix]struct{}{}
				for _, pref := range dl.V6Meta.Renewable {
					renewable[pref] = struct{}{}
				}

				deprecated := map[netip.Prefix]time.Time{}
				for _, dp := range dl.V6Meta.Deprecated {
					if dp == nil {
						continue
					}

					until, parseErr := time.Parse(time.RFC3339, dp.ValidUntil)
					if parseErr != nil {
						log.Info("dhcp: invalid v6 deprecated prefix %s: %s", dp.Prefix, parseErr)

						continue
					}

					deprecated[dp.Prefix] = until
				}

				srv6.setRestoredPrefixMeta(renewable, deprecated)
			}
		}
	}

	log.Info(
		"dhcp: loaded leases v4:%d  v6:%d  total-read:%d from DB",
		len(leases4),
		len(leases6),
		len(leases),
	)

	return nil
}

// dbStore stores DHCP leases.
func (s *server) dbStore() (err error) {
	// Use an empty slice here as opposed to nil so that it doesn't write
	// "null" into the database file if leases are empty.
	leases := []*dbLease{}
	var v6Meta *dataLeasesV6Meta

	for _, l := range s.srv4.getLeasesRef() {
		leases = append(leases, fromLease(l))
	}

	if s.srv6 != nil {
		if srv6, ok := s.srv6.(*v6Server); ok {
			leases6, renewable, deprecated := srv6.dbSnapshot(time.Now())
			for _, l := range leases6 {
				leases = append(leases, fromLease(l))
			}
			if len(renewable) > 0 || len(deprecated) > 0 {
				v6Meta = &dataLeasesV6Meta{
					Renewable: make([]netip.Prefix, 0, len(renewable)),
				}
				for pref := range renewable {
					v6Meta.Renewable = append(v6Meta.Renewable, pref)
				}
				slices.SortFunc(v6Meta.Renewable, prefixCompare)

				if len(deprecated) > 0 {
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
				}
			}
		} else {
			for _, l := range s.srv6.getLeasesRef() {
				leases = append(leases, fromLease(l))
			}
		}
	}

	return writeDB(s.conf.dbFilePath, leases, v6Meta)
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
