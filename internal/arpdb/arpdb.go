// Package arpdb implements the Network Neighborhood Database.
package arpdb

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/osutil"
)

// Variables and functions to substitute in tests.
var (
	// aghosRunCommand is the function to run shell commands.
	aghosRunCommand = aghos.RunCommand

	// rootDirFS is the filesystem pointing to the root directory.
	rootDirFS = osutil.RootDirFS()
)

// Interface stores and refreshes the network neighborhood reported by ARP
// (Address Resolution Protocol).
type Interface interface {
	// Refresh updates the stored data.  It must be safe for concurrent use.
	Refresh() (err error)

	// Neighbors returnes the last set of data reported by ARP.  Both the method
	// and it's result must be safe for concurrent use.
	Neighbors() (ns []Neighbor)
}

// New returns the [Interface] properly initialized for the OS.
func New(logger *slog.Logger) (arp Interface) {
	return newARPDB(logger)
}

// Empty is the [Interface] implementation that does nothing.
type Empty struct{}

// type check
var _ Interface = Empty{}

// Refresh implements the [Interface] interface for EmptyARPContainer.  It does
// nothing and always returns nil error.
func (Empty) Refresh() (err error) { return nil }

// Neighbors implements the [Interface] interface for EmptyARPContainer.  It
// always returns nil.
func (Empty) Neighbors() (ns []Neighbor) { return nil }

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

// newNeighbor returns the new initialized [Neighbor] by parsing string
// representations of IP and MAC addresses.
func newNeighbor(host, ipStr, macStr string) (n *Neighbor, err error) {
	defer func() { err = errors.Annotate(err, "getting arp neighbor: %w") }()

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		// Don't wrap the error, as it will get annotated.
		return nil, err
	}

	mac, err := net.ParseMAC(macStr)
	if err != nil {
		// Don't wrap the error, as it will get annotated.
		return nil, err
	}

	return &Neighbor{
		Name: host,
		IP:   ip,
		MAC:  mac,
	}, nil
}

// Clone returns the deep copy of n.
func (n Neighbor) Clone() (clone Neighbor) {
	return Neighbor{
		Name: n.Name,
		IP:   n.IP,
		MAC:  slices.Clone(n.MAC),
	}
}

// validatedHostname returns h if it's a valid hostname, or an empty string
// otherwise, logging the validation error.
func validatedHostname(logger *slog.Logger, h string) (host string) {
	err := netutil.ValidateHostname(h)
	if err != nil {
		logger.Debug("parsing host of arp output", slogutil.KeyError, err)

		return ""
	}

	return h
}

// neighs is the helper type that stores neighbors to avoid copying its methods
// among all the [Interface] implementations.
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

// parseNeighsFunc parses the text from sc as if it'd be an output of some
// ARP-related command.  lenHint is a hint for the size of the allocated slice
// of Neighbors.
//
// TODO(s.chzhen):  Return []*Neighbor instead.
type parseNeighsFunc func(logger *slog.Logger, sc *bufio.Scanner, lenHint int) (ns []Neighbor)

// cmdARPDB is the implementation of the [Interface] that uses command line to
// retrieve data.
type cmdARPDB struct {
	logger *slog.Logger
	parse  parseNeighsFunc
	ns     *neighs
	cmd    string
	args   []string
}

// type check
var _ Interface = (*cmdARPDB)(nil)

// Refresh implements the [Interface] interface for *cmdARPDB.
func (arp *cmdARPDB) Refresh() (err error) {
	defer func() { err = errors.Annotate(err, "cmd arpdb: %w") }()

	code, out, err := aghosRunCommand(arp.cmd, arp.args...)
	if err != nil {
		return fmt.Errorf("running command: %w", err)
	} else if code != 0 {
		return fmt.Errorf("running command: unexpected exit code %d", code)
	}

	sc := bufio.NewScanner(bytes.NewReader(out))
	ns := arp.parse(arp.logger, sc, arp.ns.len())
	if err = sc.Err(); err != nil {
		// TODO(e.burkov):  This error seems unreachable.  Investigate.
		return fmt.Errorf("scanning the output: %w", err)
	}

	arp.ns.reset(ns)

	return nil
}

// Neighbors implements the [Interface] interface for *cmdARPDB.
func (arp *cmdARPDB) Neighbors() (ns []Neighbor) {
	return arp.ns.clone()
}

// arpdbs is the [Interface] that combines several [Interface] implementations
// and consequently switches between those.
type arpdbs struct {
	// arps is the set of [Interface] implementations to range through.
	arps []Interface
	neighs
}

// newARPDBs returns a properly initialized *arpdbs.  It begins refreshing from
// the first of arps.
func newARPDBs(arps ...Interface) (arp *arpdbs) {
	return &arpdbs{
		arps: arps,
		neighs: neighs{
			mu: &sync.RWMutex{},
			ns: make([]Neighbor, 0),
		},
	}
}

// type check
var _ Interface = (*arpdbs)(nil)

// Refresh implements the [Interface] interface for *arpdbs.
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

	return errors.Annotate(errors.Join(errs...), "each arpdb failed: %w")
}

// Neighbors implements the [Interface] interface for *arpdbs.
//
// TODO(e.burkov):  Think of a way to avoid cloning the slice twice.
func (arp *arpdbs) Neighbors() (ns []Neighbor) {
	return arp.clone()
}
