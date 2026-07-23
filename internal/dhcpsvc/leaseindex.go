package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"strings"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// leaseIndex is the set of leases indexed by their identifiers for quick
// lookup.
//
// TODO(e.burkov):  Use for all lease-related operations, including
// interface-specific ones.
type leaseIndex struct {
	// byAddr is a lookup shortcut for leases by their IP addresses.
	byAddr map[netip.Addr]*Lease

	// byName is a lookup shortcut for leases by their hostnames.
	//
	// TODO(e.burkov):  Use a slice of leases with the same hostname?
	byName map[string]*Lease

	// database is the leases storage.
	database Database
}

// newLeaseIndex returns a new index for [Lease]s.
func newLeaseIndex(db Database) (idx *leaseIndex) {
	return &leaseIndex{
		byAddr:   map[netip.Addr]*Lease{},
		byName:   map[string]*Lease{},
		database: db,
	}
}

// leaseByAddr returns a lease by its IP address.
func (idx *leaseIndex) leaseByAddr(addr netip.Addr) (l *Lease, ok bool) {
	l, ok = idx.byAddr[addr]

	return l, ok
}

// leaseByName returns a lease by its hostname.
func (idx *leaseIndex) leaseByName(name string) (l *Lease, ok bool) {
	// TODO(e.burkov):  Probably, use a case-insensitive comparison and store in
	// slice.  This would require a benchmark.
	l, ok = idx.byName[strings.ToLower(name)]

	return l, ok
}

// clear removes all leases from idx.  It doesn't clear interfaces' leases.
func (idx *leaseIndex) clear(ctx context.Context) (err error) {
	clear(idx.byAddr)
	clear(idx.byName)

	err = idx.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return nil
}

// add adds l into idx and into iface.  l must be valid, iface should be
// responsible for l's IP.  It returns an error if l duplicates at least a
// single value of another lease.  It doesn't store the leases into the
// database, it's caller's responsibility to do so.
//
// TODO(e.burkov):  Support empty hostnames.
func (idx *leaseIndex) add(l *Lease, iface *netInterface) (err error) {
	loweredName := strings.ToLower(l.Hostname)

	if _, ok := idx.byAddr[l.IP]; ok {
		return fmt.Errorf("lease for ip %s already exists", l.IP)
	} else if _, ok = idx.byName[loweredName]; ok {
		return fmt.Errorf("lease for hostname %s already exists", l.Hostname)
	}

	err = iface.addLease(l)
	if err != nil {
		return err
	}

	idx.byAddr[l.IP] = l
	idx.byName[loweredName] = l

	return nil
}

// remove removes l from idx and from iface.  l must be valid, iface should
// contain the same lease or the lease itself.  It returns an error if the lease
// not found.
//
// TODO(e.burkov):  Consider using the iface's logger after simplifying
// relations between index and interfaces.
func (idx *leaseIndex) remove(
	ctx context.Context,
	l *Lease,
	iface *netInterface,
) (err error) {
	loweredName := strings.ToLower(l.Hostname)

	if _, ok := idx.byAddr[l.IP]; !ok {
		return fmt.Errorf("no lease for ip %s", l.IP)
	} else if _, ok = idx.byName[loweredName]; !ok {
		return fmt.Errorf("no lease for hostname %s", l.Hostname)
	}

	err = iface.removeLease(l)
	if err != nil {
		return err
	}

	delete(idx.byAddr, l.IP)
	delete(idx.byName, loweredName)

	err = idx.dbStore(ctx)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return nil
}

// update updates l in idx and in iface.  l must be valid, iface should be
// responsible for l's IP.  It returns an error if l duplicates at least a
// single value of another lease, except for the updated lease itself.
//
// TODO(e.burkov):  Support empty hostnames.
func (idx *leaseIndex) update(
	ctx context.Context,
	l *Lease,
	iface *netInterface,
) (err error) {
	loweredName := strings.ToLower(l.Hostname)

	existing, ok := idx.byAddr[l.IP]
	if ok && !slices.Equal(l.HWAddr, existing.HWAddr) {
		return fmt.Errorf("lease for ip %s already exists", l.IP)
	}

	existing, ok = idx.byName[loweredName]
	if ok && !slices.Equal(l.HWAddr, existing.HWAddr) {
		return fmt.Errorf("lease for hostname %s already exists", l.Hostname)
	}

	prev, err := iface.updateLease(l)
	if err != nil {
		return err
	}

	delete(idx.byAddr, prev.IP)
	delete(idx.byName, strings.ToLower(prev.Hostname))

	idx.byAddr[l.IP] = l
	idx.byName[loweredName] = l

	return idx.dbStore(ctx)
}

// rangeLeases calls f for each lease in idx in an unspecified order until f
// returns false.  It must not be called concurrently, f must not modify leases.
func (idx *leaseIndex) rangeLeases(f func(l *Lease) (cont bool)) {
	for _, l := range idx.byName {
		if !f(l) {
			break
		}
	}
}

// dbLoad loads stored leases.  It must only be called before the service has
// been started.
func (idx *leaseIndex) dbLoad(
	ctx context.Context,
	logger *slog.Logger,
	ifaces4 dhcpInterfacesV4,
	ifaces6 dhcpInterfacesV6,
) (err error) {
	leases, err := idx.database.Load(ctx)
	if err != nil {
		return fmt.Errorf("loading leases: %w", err)
	}

	idx.addDBLeases(ctx, logger, leases, ifaces4, ifaces6)

	return nil
}

// addDBLeases adds leases to the server.  logger must not be nil.
func (idx *leaseIndex) addDBLeases(
	ctx context.Context,
	logger *slog.Logger,
	leases []*Lease,
	ifaces4 dhcpInterfacesV4,
	ifaces6 dhcpInterfacesV6,
) {
	var v4, v6 uint
	for i, l := range leases {
		iface, err := ifaceForAddr(l.IP, ifaces4, ifaces6)
		if err != nil {
			logger.WarnContext(ctx, "searching lease iface", "idx", i, slogutil.KeyError, err)

			continue
		}

		err = idx.add(l.Clone(), iface)
		if err != nil {
			logger.WarnContext(ctx, "adding lease", "idx", i, slogutil.KeyError, err)

			continue
		}

		if l.IP.Is4() {
			v4++
		} else {
			v6++
		}
	}

	// TODO(e.burkov):  Group by interface.
	logger.InfoContext(ctx, "loaded leases", "v4", v4, "v6", v6, "total", len(leases))
}

// dbStore writes leases to the database file.  The [DHCPServer.leasesMu] must
// be locked.
func (idx *leaseIndex) dbStore(ctx context.Context) (err error) {
	leases := make([]*Lease, 0, len(idx.byAddr))
	for _, l := range idx.byAddr {
		leases = append(leases, l)
	}

	err = idx.database.Store(ctx, leases)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return nil
}
