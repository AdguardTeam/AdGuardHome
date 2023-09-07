//go:build linux

package arpdb

import (
	"bufio"
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
)

func newARPDB() (arp *arpdbs) {
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
			parse: parseF,
			ns:    ns,
			cmd:   "arp",
			// Use -n flag to avoid resolving the hostnames of the neighbors.
			// By default ARP attempts to resolve the hostnames via DNS.  See
			// man 8 arp.
			//
			// See also https://github.com/AdguardTeam/AdGuardHome/issues/3157.
			args: []string{"-a", "-n"},
		},
		// Finally, try "ip neigh".
		&cmdARPDB{
			parse: parseIPNeigh,
			ns:    ns,
			cmd:   "ip",
			args:  []string{"neigh"},
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
func parseArpAWrt(sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
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

		ip, err := netip.ParseAddr(fields[0])
		if err != nil {
			log.Debug("arpdb: parsing arp output: ip: %s", err)

			continue
		}

		hwStr := fields[3]
		mac, err := net.ParseMAC(hwStr)
		if err != nil {
			log.Debug("arpdb: parsing arp output: mac: %s", err)

			continue
		}

		ns = append(ns, Neighbor{
			IP:  ip,
			MAC: mac,
		})
	}

	return ns
}

// parseArpA parses the output of the "arp -a -n" command on Linux.  The
// expected input format:
//
//	hostname (192.168.1.1) at ab:cd:ef:ab:cd:ef [ether] on enp0s3
func parseArpA(sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
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

		ip, err := netip.ParseAddr(ipStr[1 : len(ipStr)-1])
		if err != nil {
			log.Debug("arpdb: parsing arp output: ip: %s", err)

			continue
		}

		hwStr := fields[3]
		mac, err := net.ParseMAC(hwStr)
		if err != nil {
			log.Debug("arpdb: parsing arp output: mac: %s", err)

			continue
		}

		ns = append(ns, Neighbor{
			IP:   ip,
			MAC:  mac,
			Name: validatedHostname(fields[0]),
		})
	}

	return ns
}

// parseIPNeigh parses the output of the "ip neigh" command on Linux.  The
// expected input format:
//
//	192.168.1.1 dev enp0s3 lladdr ab:cd:ef:ab:cd:ef REACHABLE
func parseIPNeigh(sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()

		fields := strings.Fields(ln)
		if len(fields) < 5 {
			continue
		}

		n := Neighbor{}

		ip, err := netip.ParseAddr(fields[0])
		if err != nil {
			log.Debug("arpdb: parsing arp output: ip: %s", err)

			continue
		} else {
			n.IP = ip
		}

		mac, err := net.ParseMAC(fields[4])
		if err != nil {
			log.Debug("arpdb: parsing arp output: mac: %s", err)

			continue
		} else {
			n.MAC = mac
		}

		ns = append(ns, n)
	}

	return ns
}
