package dhcpsvc

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// leaseIndex is the set of leases indexed by their identifiers for quick
// lookup.
type leaseIndex struct {
	// byAddr is a lookup shortcut for leases by their IP addresses.
	byAddr map[netip.Addr]*Lease

	// byName is a lookup shortcut for leases by their hostnames.
	//
	// TODO(e.burkov):  Use a slice of leases with the same hostname?
	byName map[string]*Lease
}

// newLeaseIndex returns a new index for [Lease]s.
func newLeaseIndex() *leaseIndex {
	return &leaseIndex{
		byAddr: map[netip.Addr]*Lease{},
		byName: map[string]*Lease{},
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

// clear removes all leases from idx.
func (idx *leaseIndex) clear() {
	clear(idx.byAddr)
	clear(idx.byName)
}

// add adds l into idx and into iface.  l must be valid, iface should be
// responsible for l's IP.  It returns an error if l duplicates at least a
// single value of another lease.
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
func (idx *leaseIndex) remove(l *Lease, iface *netInterface) (err error) {
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

	return nil
}

// update updates l in idx and in iface.  l must be valid, iface should be
// responsible for l's IP.  It returns an error if l duplicates at least a
// single value of another lease, except for the updated lease itself.
func (idx *leaseIndex) update(l *Lease, iface *netInterface) (err error) {
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

	return nil
}

// rangeLeases calls f for each lease in idx in an unspecified order until f
// returns false.
func (idx *leaseIndex) rangeLeases(f func(l *Lease) (cont bool)) {
	for _, l := range idx.byName {
		if !f(l) {
			break
		}
	}
}

// len returns the number of leases in idx.
func (idx *leaseIndex) len() (l uint) {
	return uint(len(idx.byAddr))
}
