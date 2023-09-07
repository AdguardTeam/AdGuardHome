//go:build openbsd

package arpdb

import (
	"bufio"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
)

func newARPDB() (arp *cmdARPDB) {
	return &cmdARPDB{
		parse: parseArpA,
		ns: &neighs{
			mu: &sync.RWMutex{},
			ns: make([]Neighbor, 0),
		},
		cmd: "arp",
		// Use -n flag to avoid resolving the hostnames of the neighbors.  By
		// default ARP attempts to resolve the hostnames via DNS.  See man 8
		// arp.
		//
		// See also https://github.com/AdguardTeam/AdGuardHome/issues/3157.
		args: []string{"-a", "-n"},
	}
}

// parseArpA parses the output of the "arp -a -n" command on OpenBSD.  The
// expected input format:
//
//	Host        Ethernet Address  Netif Expire    Flags
//	192.168.1.1 ab:cd:ef:ab:cd:ef   em0 19m59s
func parseArpA(sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	// Skip the header.
	if !sc.Scan() {
		return nil
	}

	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()

		fields := strings.Fields(ln)
		if len(fields) < 2 {
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

		mac, err := net.ParseMAC(fields[1])
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
