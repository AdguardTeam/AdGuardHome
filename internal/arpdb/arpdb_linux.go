//go:build linux

package arpdb

import (
	"bufio"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/stringutil"
)

func newARPDB(logger *slog.Logger) (arp *arpdbs) {
	// Use the common storage among the implementations.
	ns := &neighs{
		mu: &sync.RWMutex{},
		ns: make([]Neighbor, 0),
	}

	var parseF parseNeighsFunc
	if aghos.IsOpenWrt() {
		parseF = parseArpAWrt
	} else {
		parseF = parseArpA
	}

	return newARPDBs(
		// Try /proc/net/arp first.
		&fsysARPDB{
			ns:       ns,
			fsys:     rootDirFS,
			filename: "proc/net/arp",
		},
		// Then, try "arp -a -n".
		&cmdARPDB{
			logger: logger,
			parse:  parseF,
			ns:     ns,
			cmd:    "arp",
			// Use -n flag to avoid resolving the hostnames of the neighbors.
			// By default ARP attempts to resolve the hostnames via DNS.  See
			// man 8 arp.
			//
			// See also https://github.com/AdguardTeam/AdGuardHome/issues/3157.
			args: []string{"-a", "-n"},
		},
		// Finally, try "ip neigh".
		&cmdARPDB{
			logger: logger,
			parse:  parseIPNeigh,
			ns:     ns,
			cmd:    "ip",
			args:   []string{"neigh"},
		},
	)
}

// fsysARPDB accesses the ARP cache file to update the database.
type fsysARPDB struct {
	ns       *neighs
	fsys     fs.FS
	filename string
}

// type check
var _ Interface = (*fsysARPDB)(nil)

// Refresh implements the [Interface] interface for *fsysARPDB.
func (arp *fsysARPDB) Refresh() (err error) {
	var f fs.File
	f, err = arp.fsys.Open(arp.filename)
	if err != nil {
		return fmt.Errorf("opening %q: %w", arp.filename, err)
	}

	sc := bufio.NewScanner(f)
	// Skip the header.
	if !sc.Scan() {
		return nil
	} else if err = sc.Err(); err != nil {
		return err
	}

	ns := make([]Neighbor, 0, arp.ns.len())
	for sc.Scan() {
		n := parseNeighbor(sc.Text())
		if n != nil {
			ns = append(ns, *n)
		}
	}

	arp.ns.reset(ns)

	return nil
}

// parseNeighbor parses line into *Neighbor.
func parseNeighbor(line string) (n *Neighbor) {
	fields := stringutil.SplitTrimmed(line, " ")
	if len(fields) != 6 {
		return nil
	}

	ip, err := netip.ParseAddr(fields[0])
	if err != nil || ip.IsUnspecified() {
		return nil
	}

	mac, err := net.ParseMAC(fields[3])
	if err != nil {
		return nil
	}

	return &Neighbor{
		IP:  ip,
		MAC: mac,
	}
}

// Neighbors implements the [Interface] interface for *fsysARPDB.
func (arp *fsysARPDB) Neighbors() (ns []Neighbor) {
	return arp.ns.clone()
}

// parseArpAWrt parses the output of the "arp -a -n" command on OpenWrt.  The
// expected input format:
//
//	IP address     HW type  Flags  HW address         Mask  Device
//	192.168.11.98  0x1      0x2    5a:92:df:a9:7e:28  *     wan
func parseArpAWrt(logger *slog.Logger, sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	if !sc.Scan() {
		// Skip the header.
		return
	}

	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()

		fields := strings.Fields(ln)
		if len(fields) < 4 {
			continue
		}

		n, err := newNeighbor("", fields[0], fields[3])
		if err != nil {
			logger.Debug("parsing arp output", "line", ln, slogutil.KeyError, err)

			continue
		}

		ns = append(ns, *n)
	}

	return ns
}

// parseArpA parses the output of the "arp -a -n" command on Linux.  The
// expected input format:
//
//	hostname (192.168.1.1) at ab:cd:ef:ab:cd:ef [ether] on enp0s3
func parseArpA(logger *slog.Logger, sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()

		fields := strings.Fields(ln)
		if len(fields) < 4 {
			continue
		}

		ipStr := fields[1]
		if len(ipStr) < 2 {
			continue
		}

		host := validatedHostname(logger, fields[0])
		n, err := newNeighbor(host, ipStr[1:len(ipStr)-1], fields[3])
		if err != nil {
			logger.Debug("parsing arp output", "line", ln, slogutil.KeyError, err)

			continue
		}

		ns = append(ns, *n)
	}

	return ns
}

// parseIPNeigh parses the output of the "ip neigh" command on Linux.  The
// expected input format:
//
//	192.168.1.1 dev enp0s3 lladdr ab:cd:ef:ab:cd:ef REACHABLE
func parseIPNeigh(logger *slog.Logger, sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()

		fields := strings.Fields(ln)
		if len(fields) < 5 {
			continue
		}

		n, err := newNeighbor("", fields[0], fields[4])
		if err != nil {
			logger.Debug("parsing arp output", "line", ln, slogutil.KeyError, err)

			continue
		}

		ns = append(ns, *n)
	}

	return ns
}
