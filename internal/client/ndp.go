package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/timeutil"
)

const (
	// ndpDataTTL is the time after which the data read from the IPv6 neighbor
	// table is considered outdated.  It's also the minimum interval between two
	// reads of the table, which limits the number of the processes spawned on
	// the DNS request path.
	ndpDataTTL = 30 * time.Second

	// ndpReadTimeout is the maximum time to spend reading the IPv6 neighbor
	// table.
	ndpReadTimeout = 5 * time.Second
)

// ndpNeighbors maps the IPv6 addresses of the network neighbors to their MAC
// addresses, as reported by NDP (Neighbor Discovery Protocol).  It's used to
// identify persistent clients by MAC address when their requests come from IPv6
// addresses that have no DHCP lease, e.g. the ones configured via SLAAC.
//
// A nil *ndpNeighbors reports no neighbors, which is the case on the platforms
// where reading the neighbor table isn't supported.
type ndpNeighbors struct {
	// logger is used for logging the operation of the reader.  It must not be
	// nil.
	logger *slog.Logger

	// clock is used to check how long ago the neighbor table was read.  It must
	// not be nil.
	clock timeutil.Clock

	// cmdCons is used to run the command that prints the neighbor table.  It
	// must not be nil.
	cmdCons executil.CommandConstructor

	// mu protects all the fields below.
	mu *sync.RWMutex

	// neighbors maps the IPv6 addresses of the known neighbors to their MAC
	// addresses.  Link-local addresses include their zone.
	neighbors map[netip.Addr]net.HardwareAddr

	// updated is the time of the last successful read of the neighbor table.
	updated time.Time

	// attempted is the time of the last read of the neighbor table, successful
	// or not.
	attempted time.Time

	// isReading is true while the neighbor table is being read, so that only a
	// single read runs at a time.
	isReading bool
}

// newNDPNeighbors returns a new properly initialized *ndpNeighbors.  logger,
// clock, and cmdCons must not be nil.  It returns nil if reading the IPv6
// neighbor table isn't supported on the current platform.
//
// TODO(AndyHazz):  Support "ndp -an" on BSD and Darwin, and "netsh interface
// ipv6 show neighbors" on Windows.
func newNDPNeighbors(
	logger *slog.Logger,
	clock timeutil.Clock,
	cmdCons executil.CommandConstructor,
) (n *ndpNeighbors) {
	if runtime.GOOS != "linux" {
		return nil
	}

	return &ndpNeighbors{
		logger:    logger,
		clock:     clock,
		cmdCons:   cmdCons,
		mu:        &sync.RWMutex{},
		neighbors: map[netip.Addr]net.HardwareAddr{},
	}
}

// macFor returns the MAC address of the neighbor with the IPv6 address ip, if
// it's known.  ip must contain the zone, if it's a link-local address.
//
// If ip is unknown or the data has become outdated, macFor schedules a read of
// the neighbor table and returns the data that it currently has, if any.  It
// doesn't wait for the read to finish, since it's called on the DNS request
// path, where the callers may hold their locks.
func (n *ndpNeighbors) macFor(ip netip.Addr) (mac net.HardwareAddr) {
	if n == nil || !ip.Is6() || ip.Is4In6() {
		return nil
	}

	n.mu.RLock()
	mac = n.neighbors[ip]
	isOutdated := n.clock.Now().Sub(n.updated) >= ndpDataTTL
	n.mu.RUnlock()

	if mac == nil || isOutdated {
		n.scheduleRead()
	}

	return mac
}

// refresh reads the IPv6 neighbor table and replaces the known neighbors with
// the data from it.  It does nothing if the table is already being read, and
// keeps the previously known neighbors if the read fails.
func (n *ndpNeighbors) refresh(ctx context.Context) {
	if n == nil || !n.beginRead(true) {
		return
	}

	defer n.endRead()

	n.read(ctx)
}

// scheduleRead reads the IPv6 neighbor table in a separate goroutine, unless
// it's being read right now or has been read within [ndpDataTTL].
//
// TODO(AndyHazz):  Pass the context once the client lookup methods accept it.
func (n *ndpNeighbors) scheduleRead() {
	if !n.beginRead(false) {
		return
	}

	go func() {
		ctx := context.TODO()

		defer slogutil.RecoverAndLog(ctx, n.logger)
		defer n.endRead()

		n.read(ctx)
	}()
}

// beginRead reports whether the caller should read the IPv6 neighbor table,
// marking the read as started if it should.  Unless force is true, the read is
// also skipped if the table has been read within [ndpDataTTL].  The caller must
// call [ndpNeighbors.endRead] once the read is finished.
func (n *ndpNeighbors) beginRead(force bool) (ok bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := n.clock.Now()
	if n.isReading || (!force && now.Sub(n.attempted) < ndpDataTTL) {
		return false
	}

	n.isReading = true
	n.attempted = now

	return true
}

// endRead marks the read of the IPv6 neighbor table as finished.
func (n *ndpNeighbors) endRead() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.isReading = false
}

// read reads the IPv6 neighbor table and replaces the known neighbors with the
// data from it.  On error, the previously known neighbors are kept, since they
// are still more useful than nothing.
func (n *ndpNeighbors) read(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, ndpReadTimeout)
	defer cancel()

	data, err := n.readTable(ctx)
	if err != nil {
		n.logger.DebugContext(ctx, "reading ipv6 neighbors", slogutil.KeyError, err)

		return
	}

	neighbors := parseNDPNeighbors(data)

	n.mu.Lock()
	defer n.mu.Unlock()

	n.neighbors = neighbors
	n.updated = n.clock.Now()
}

// readTable returns the output of the command that prints the IPv6 neighbor
// table.
func (n *ndpNeighbors) readTable(ctx context.Context) (data []byte, err error) {
	defer func() { err = errors.Annotate(err, "ip -6 neigh: %w") }()

	var stdout bytes.Buffer
	err = executil.Run(ctx, n.cmdCons, &executil.CommandConfig{
		Path:   "ip",
		Args:   []string{"-6", "neigh"},
		Stdout: &stdout,
	})
	if err != nil {
		if code, ok := executil.ExitCodeFromError(err); ok {
			return nil, fmt.Errorf("unexpected exit code %d", code)
		}

		// Don't wrap the error, as it will get annotated.
		return nil, err
	}

	return stdout.Bytes(), nil
}

// parseNDPNeighbors parses the output of the "ip -6 neigh" command.  The
// expected input format:
//
//	2001:db8::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE
//	fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff router STALE
//	2001:db8::2 dev eth0 FAILED
func parseNDPNeighbors(data []byte) (ns map[netip.Addr]net.HardwareAddr) {
	ns = map[netip.Addr]net.HardwareAddr{}
	sc := bufio.NewScanner(bytes.NewReader(data))

	for sc.Scan() {
		ip, mac := parseNDPNeighbor(strings.Fields(sc.Text()))
		if mac != nil {
			ns[ip] = mac
		}
	}

	return ns
}

// parseNDPNeighbor parses a single line of the "ip -6 neigh" output, split into
// fields.  mac is nil if the line doesn't contain a valid pair of an IPv6
// address and a link-layer address, which is the case for the incomplete
// entries.
//
// Link-local addresses are only unique within their link, so they are returned
// with the interface from the "dev" field as their zone, just like the addresses
// of the incoming DNS requests.
func parseNDPNeighbor(fields []string) (ip netip.Addr, mac net.HardwareAddr) {
	// The address is followed by at least the "dev" and "lladdr" fields with
	// their values.
	if len(fields) < 5 {
		return netip.Addr{}, nil
	}

	ip, err := netip.ParseAddr(fields[0])
	if err != nil {
		return netip.Addr{}, nil
	}

	dev, macStr := ndpNeighborFields(fields[1:])
	if macStr == "" {
		return netip.Addr{}, nil
	}

	mac, err = net.ParseMAC(macStr)
	if err != nil {
		return netip.Addr{}, nil
	}

	if dev != "" && ip.IsLinkLocalUnicast() {
		ip = ip.WithZone(dev)
	}

	return ip, mac
}

// ndpNeighborFields returns the values of the "dev" and "lladdr" fields of a
// single entry of the IPv6 neighbor table.  Those that aren't present are
// returned empty.  fields must not be empty.
func ndpNeighborFields(fields []string) (dev, mac string) {
	for i, f := range fields[:len(fields)-1] {
		switch f {
		case "dev":
			dev = fields[i+1]
		case "lladdr":
			mac = fields[i+1]
		}
	}

	return dev, mac
}
