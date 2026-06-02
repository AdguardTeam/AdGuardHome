package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
)

// macKey contains hardware address as byte array of 6, 8, or 20 bytes.
//
// TODO(e.burkov):  Move to aghnet or even to netutil.
//
// TODO(e.burkov):  Identify the client by the hardware address and the client
// identifier from the DHCP messages.
//
// TODO(e.burkov):  Identify IPv6 clients with DUID.
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
// [Lease.IsBlocked].  It also removes the lease from iface, but leaves it in
// the index.  iface.indexMu must be locked, l and clock must not be nil.
func (iface *netInterface) blockLease(
	ctx context.Context,
	l *Lease,
	clock timeutil.Clock,
) (err error) {
	err = iface.removeLease(l)
	if err != nil {
		return fmt.Errorf("removing lease: %w", err)
	}

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

// nextIP generates a new free IP.  It returns netip.Addr{} if there are no free
// IPs in the address space.  iface.indexMu must be locked.
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

// findExpiredLease returns the first found lease that has expired.
// iface.indexMu must be locked.
func (iface *netInterface) findExpiredLease(now time.Time) (l *Lease) {
	for _, lease := range iface.leases {
		if !lease.IsStatic && lease.Expiry.Before(now) {
			return lease
		}
	}

	return nil
}

// allocateLease allocates a new lease for the MAC address.  If there are no IP
// addresses left, both lease and err are nil.  mac must be a valid according to
// [netutil.ValidateMAC].
//
// TODO(e.burkov):  Pass the precalculated macKey.
func (iface *netInterface) allocateLease(
	ctx context.Context,
	mac net.HardwareAddr,
	checker addressChecker,
	clock timeutil.Clock,
) (lease *Lease, err error) {
	key := macToKey(mac)

	for {
		lease, err = iface.reserveLease(ctx, mac, clock)
		if err != nil {
			return nil, err
		}

		var ok bool
		ok, err = checker.IsAvailable(lease.IP)
		if err != nil {
			return nil, fmt.Errorf("checking address availability: %w", err)
		}

		if ok {
			iface.leases[key] = lease

			off, _ := iface.addrSpace.offset(lease.IP)
			iface.leasedOffsets.set(off, true)

			return lease, nil
		}

		iface.logger.DebugContext(ctx, "address not available", "ip", lease.IP)

		err = iface.blockLease(ctx, lease, clock)
		if err != nil {
			return nil, fmt.Errorf("blocking unavailable address: %w", err)
		}
	}
}

// reserveLease reserves a lease for a client by its MAC-address.  lease is nil
// if a new lease can't be allocated.  mac must be a valid according to
// [netutil.ValidateMAC].  iface.indexMu mutex must be locked.
func (iface *netInterface) reserveLease(
	ctx context.Context,
	mac net.HardwareAddr,
	clock timeutil.Clock,
) (lease *Lease, err error) {
	// TODO(e.burkov):  Limit the number of attempts.
	nextIP := iface.nextIP()
	if nextIP != (netip.Addr{}) {
		lease = &Lease{
			HWAddr: slices.Clone(mac),
			IP:     nextIP,
			Expiry: clock.Now().Add(iface.leaseTTL),
		}

		return lease, nil
	}

	lease = iface.findExpiredLease(clock.Now())
	if lease == nil {
		return nil, errors.Error("no addresses available to lease")
	}

	err = iface.index.remove(ctx, iface.logger, lease, iface)
	if err != nil {
		// TODO(e.burkov):  Reconsider the severity of this error, it actually
		// seems impossible to get the error about the existing lease from the
		// method.
		iface.logger.DebugContext(ctx, "deleting expired lease", slogutil.KeyError, err)
	}

	lease.HWAddr = slices.Clone(mac)
	lease.Hostname = ""
	lease.IsStatic = false
	lease.updateExpiry(clock, iface.leaseTTL)

	iface.leases[macToKey(mac)] = lease

	return lease, nil
}
