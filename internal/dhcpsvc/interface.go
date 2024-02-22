package dhcpsvc

import (
	"fmt"
	"slices"
	"time"
)

// netInterface is a common part of any network interface within the DHCP
// server.
//
// TODO(e.burkov):  Add other methods as [DHCPServer] evolves.
type netInterface struct {
	// name is the name of the network interface.
	name string

	// leases is a set of leases sorted by hardware address.
	leases []*Lease

	// leaseTTL is the default Time-To-Live value for leases.
	leaseTTL time.Duration
}

// reset clears all the slices in iface for reuse.
func (iface *netInterface) reset() {
	iface.leases = iface.leases[:0]
}

// insertLease inserts the given lease into iface.  It returns an error if the
// lease can't be inserted.
func (iface *netInterface) insertLease(l *Lease) (err error) {
	i, found := slices.BinarySearchFunc(iface.leases, l, compareLeaseMAC)
	if found {
		return fmt.Errorf("lease for mac %s already exists", l.HWAddr)
	}

	iface.leases = slices.Insert(iface.leases, i, l)

	return nil
}

// updateLease replaces an existing lease within iface with the given one.  It
// returns an error if there is no lease with such hardware address.
func (iface *netInterface) updateLease(l *Lease) (prev *Lease, err error) {
	i, found := slices.BinarySearchFunc(iface.leases, l, compareLeaseMAC)
	if !found {
		return nil, fmt.Errorf("no lease for mac %s", l.HWAddr)
	}

	prev, iface.leases[i] = iface.leases[i], l

	return prev, nil
}

// removeLease removes an existing lease from iface.  It returns an error if
// there is no lease equal to l.
func (iface *netInterface) removeLease(l *Lease) (err error) {
	i, found := slices.BinarySearchFunc(iface.leases, l, compareLeaseMAC)
	if !found {
		return fmt.Errorf("no lease for mac %s", l.HWAddr)
	}

	iface.leases = slices.Delete(iface.leases, i, i+1)

	return nil
}
