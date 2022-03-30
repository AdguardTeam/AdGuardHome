//go:build openbsd
// +build openbsd

package aghnet

import (
	"bufio"
	"net"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
)

func newARPDB() *cmdARPDB {
	return &cmdARPDB{
		parse: parseArpA,
		ns: &neighs{
			mu: &sync.RWMutex{},
			ns: make([]Neighbor, 0),
		},
		cmd:  "arp",
		args: []string{"-a"},
	}
}

// parseArpA parses the output of the "arp -a" command on OpenBSD.  The expected
// input format:
//
//   Host        Ethernet Address  Netif Expire    Flags
//   192.168.1.1 ab:cd:ef:ab:cd:ef   em0 19m59s
//
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

		if ip := net.ParseIP(fields[0]); ip == nil {
			continue
		} else {
			n.IP = ip
		}

		if mac, err := net.ParseMAC(fields[1]); err != nil {
			log.Debug("parsing arp output: %s", err)

			continue
		} else {
			n.MAC = mac
		}

		ns = append(ns, n)
	}

	return ns
}
