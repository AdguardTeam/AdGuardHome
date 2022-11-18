package aghnet

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/exp/slices"
)

// ARPDB: The Network Neighborhood Database

// ARPDB stores and refreshes the network neighborhood reported by ARP (Address
// Resolution Protocol).
type ARPDB interface {
	// Refresh updates the stored data.  It must be safe for concurrent use.
	Refresh() (err error)

	// Neighbors returnes the last set of data reported by ARP.  Both the method
	// and it's result must be safe for concurrent use.
	Neighbors() (ns []Neighbor)
}

// NewARPDB returns the ARPDB properly initialized for the OS.
func NewARPDB() (arp ARPDB) {
	return newARPDB()
}

// Empty ARPDB implementation

// EmptyARPDB is the ARPDB implementation that does nothing.
type EmptyARPDB struct{}

// type check
var _ ARPDB = EmptyARPDB{}

// Refresh implements the ARPDB interface for EmptyARPContainer.  It does
// nothing and always returns nil error.
func (EmptyARPDB) Refresh() (err error) { return nil }

// Neighbors implements the ARPDB interface for EmptyARPContainer.  It always
// returns nil.
func (EmptyARPDB) Neighbors() (ns []Neighbor) { return nil }

// ARPDB Helper Types

// Neighbor is the pair of IP address and MAC address reported by ARP.
type Neighbor struct {
	// Name is the hostname of the neighbor.  Empty name is valid since not each
	// implementation of ARP is able to retrieve that.
	Name string

	// IP contains either IPv4 or IPv6.
	IP netip.Addr

	// MAC contains the hardware address.
	MAC net.HardwareAddr
}

// Clone returns the deep copy of n.
func (n Neighbor) Clone() (clone Neighbor) {
	return Neighbor{
		Name: n.Name,
		IP:   n.IP,
		MAC:  slices.Clone(n.MAC),
	}
}

// neighs is the helper type that stores neighbors to avoid copying its methods
// among all the ARPDB implementations.
type neighs struct {
	mu *sync.RWMutex
	ns []Neighbor
}

// len returns the length of the neighbors slice.  It's safe for concurrent use.
func (ns *neighs) len() (l int) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return len(ns.ns)
}

// clone returns a deep copy of the underlying neighbors slice.  It's safe for
// concurrent use.
func (ns *neighs) clone() (cloned []Neighbor) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	cloned = make([]Neighbor, len(ns.ns))
	for i, n := range ns.ns {
		cloned[i] = n.Clone()
	}

	return cloned
}

// reset replaces the underlying slice with the new one.  It's safe for
// concurrent use.
func (ns *neighs) reset(with []Neighbor) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.ns = with
}

// Command ARPDB

// parseNeighsFunc parses the text from sc as if it'd be an output of some
// ARP-related command.  lenHint is a hint for the size of the allocated slice
// of Neighbors.
type parseNeighsFunc func(sc *bufio.Scanner, lenHint int) (ns []Neighbor)

// cmdARPDB is the implementation of the ARPDB that uses command line to
// retrieve data.
type cmdARPDB struct {
	parse parseNeighsFunc
	ns    *neighs
	cmd   string
	args  []string
}

// type check
var _ ARPDB = (*cmdARPDB)(nil)

// Refresh implements the ARPDB interface for *cmdARPDB.
func (arp *cmdARPDB) Refresh() (err error) {
	defer func() { err = errors.Annotate(err, "cmd arpdb: %w") }()

	code, out, err := aghosRunCommand(arp.cmd, arp.args...)
	if err != nil {
		return fmt.Errorf("running command: %w", err)
	} else if code != 0 {
		return fmt.Errorf("running command: unexpected exit code %d", code)
	}

	sc := bufio.NewScanner(bytes.NewReader(out))
	ns := arp.parse(sc, arp.ns.len())
	if err = sc.Err(); err != nil {
		// TODO(e.burkov):  This error seems unreachable.  Investigate.
		return fmt.Errorf("scanning the output: %w", err)
	}

	arp.ns.reset(ns)

	return nil
}

// Neighbors implements the ARPDB interface for *cmdARPDB.
func (arp *cmdARPDB) Neighbors() (ns []Neighbor) {
	return arp.ns.clone()
}

// Composite ARPDB

// arpdbs is the ARPDB that combines several ARPDB implementations and
// consequently switches between those.
type arpdbs struct {
	// arps is the set of ARPDB implementations to range through.
	arps []ARPDB
	neighs
}

// newARPDBs returns a properly initialized *arpdbs.  It begins refreshing from
// the first of arps.
func newARPDBs(arps ...ARPDB) (arp *arpdbs) {
	return &arpdbs{
		arps: arps,
		neighs: neighs{
			mu: &sync.RWMutex{},
			ns: make([]Neighbor, 0),
		},
	}
}

// type check
var _ ARPDB = (*arpdbs)(nil)

// Refresh implements the ARPDB interface for *arpdbs.
func (arp *arpdbs) Refresh() (err error) {
	var errs []error

	for _, a := range arp.arps {
		err = a.Refresh()
		if err != nil {
			errs = append(errs, err)

			continue
		}

		arp.reset(a.Neighbors())

		return nil
	}

	if len(errs) > 0 {
		err = errors.List("each arpdb failed", errs...)
	}

	return err
}

// Neighbors implements the ARPDB interface for *arpdbs.
//
// TODO(e.burkov):  Think of a way to avoid cloning the slice twice.
func (arp *arpdbs) Neighbors() (ns []Neighbor) {
	return arp.clone()
}
