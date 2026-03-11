package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
)

// macKey contains hardware address as byte array of 6, 8, or 20 bytes.
//
// TODO(e.burkov):  Move to aghnet or even to netutil.
type macKey any

// macToKey converts mac into macKey, which is used as the key for the lease
// maps.  mac must be a valid hardware address of length 6, 8, or 20 bytes, see
// [netutil.ValidateMAC].
func macToKey(mac net.HardwareAddr) (key macKey) {
	switch len(mac) {
	case 6:
		return [6]byte(mac)
	case 8:
		return [8]byte(mac)
	case 20:
		return [20]byte(mac)
	default:
		panic(fmt.Errorf("invalid mac address %#v", mac))
	}
}

// netInterface is a common part of any interface within the DHCP server.
//
// TODO(e.burkov):  Add other methods as [DHCPServer] evolves.
type netInterface struct {
	// logger logs the events related to the network interface.
	//
	// TODO(e.burkov):  Consider removing it and using the value from context.
	logger *slog.Logger

	// indexMu protects the index, leases, and leasedOffsets.
	indexMu *sync.RWMutex

	// leasedOffsets contains offsets from conf.ipRange.start that have been
	// leased.
	leasedOffsets *bitSet

	// index stores the DHCP leases for quick lookups.
	index *leaseIndex

	// leases is the set of DHCP leases assigned to this interface.
	leases map[macKey]*Lease

	// addrSpace is the IPv4 address space allocated for leasing.
	addrSpace ipRange

	// name is the name of the network interface.
	name string

	// leaseTTL is the default Time-To-Live value for leases.
	leaseTTL time.Duration
}

// reset clears all the slices in iface for reuse.
func (iface *netInterface) reset() {
	clear(iface.leases)
}

// addLease inserts the given lease into iface.  It returns an error if the
// lease can't be inserted.
func (iface *netInterface) addLease(l *Lease) (err error) {
	mk := macToKey(l.HWAddr)
	_, found := iface.leases[mk]
	if found {
		return fmt.Errorf("lease for mac %s already exists", l.HWAddr)
	}

	iface.leases[mk] = l

	off, _ := iface.addrSpace.offset(l.IP)
	iface.leasedOffsets.set(off, true)

	return nil
}

// updateLease replaces an existing lease within iface with the given one.  It
// returns an error if there is no lease with such hardware address.
func (iface *netInterface) updateLease(l *Lease) (prev *Lease, err error) {
	mk := macToKey(l.HWAddr)
	prev, found := iface.leases[mk]
	if !found {
		return nil, fmt.Errorf("no lease for mac %s", l.HWAddr)
	}

	iface.leases[mk] = l

	return prev, nil
}

// removeLease removes an existing lease from iface.  It returns an error if
// there is no lease equal to l.  l must not be nil.
func (iface *netInterface) removeLease(l *Lease) (err error) {
	mk := macToKey(l.HWAddr)
	_, found := iface.leases[mk]
	if !found {
		return fmt.Errorf("no lease for mac %s", l.HWAddr)
	}

	delete(iface.leases, mk)

	off, _ := iface.addrSpace.offset(l.IP)
	iface.leasedOffsets.set(off, false)

	return nil
}

// blockLease marks l as blocked for a configured TTL, as reported by
// [Lease.IsBlocked].  indexMu must be locked,  It also removes the lease from
// iface.  l must not be nil.
func (iface *netInterface) blockLease(
	ctx context.Context,
	l *Lease,
	clock timeutil.Clock,
) (err error) {
	l.HWAddr = blockedHardwareAddr
	l.Hostname = ""
	l.Expiry = clock.Now().Add(iface.leaseTTL)
	l.IsStatic = false

	err = iface.index.dbStore(ctx, iface.logger)
	if err != nil {
		return fmt.Errorf("storing index: %w", err)
	}

	return nil
}

// nextIP generates a new free IP.
func (iface *netInterface) nextIP() (ip netip.Addr) {
	r := iface.addrSpace
	ip = r.find(func(next netip.Addr) (ok bool) {
		offset, ok := r.offset(next)
		if !ok {
			panic(fmt.Errorf("next: %s: %w", next, errors.ErrOutOfRange))
		}

		return !iface.leasedOffsets.isSet(offset)
	})

	return ip
}

// findExpiredLease returns the first found lease that has expired.  indexMu
// must be locked.
func (iface *netInterface) findExpiredLease(now time.Time) (l *Lease) {
	for _, lease := range iface.leases {
		if !lease.IsStatic && lease.Expiry.Before(now) {
			return lease
		}
	}

	return nil
}
