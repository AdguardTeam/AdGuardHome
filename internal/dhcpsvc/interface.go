package dhcpsvc

import (
	"fmt"
	"log/slog"
	"net"
	"time"
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
	logger *slog.Logger

	// leases is the set of DHCP leases assigned to this interface.
	leases map[macKey]*Lease

	// name is the name of the network interface.
	name string

	// leaseTTL is the default Time-To-Live value for leases.
	leaseTTL time.Duration
}

// newNetInterface creates a new netInterface with the given name, leaseTTL, and
// logger.
func newNetInterface(name string, l *slog.Logger, leaseTTL time.Duration) (iface *netInterface) {
	return &netInterface{
		logger:   l,
		leases:   map[macKey]*Lease{},
		name:     name,
		leaseTTL: leaseTTL,
	}
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
// there is no lease equal to l.
func (iface *netInterface) removeLease(l *Lease) (err error) {
	mk := macToKey(l.HWAddr)
	_, found := iface.leases[mk]
	if !found {
		return fmt.Errorf("no lease for mac %s", l.HWAddr)
	}

	delete(iface.leases, mk)

	return nil
}
