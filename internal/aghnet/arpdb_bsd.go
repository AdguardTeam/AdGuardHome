//go:build darwin || freebsd
// +build darwin freebsd

package aghnet

import (
	"bufio"
	"net"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
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

// parseArpA parses the output of the "arp -a -n" command on macOS and FreeBSD.
// The expected input format:
//
//   host.name (192.168.0.1) at ff:ff:ff:ff:ff:ff on en0 ifscope [ethernet]
//
func parseArpA(sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()

		fields := strings.Fields(ln)
		if len(fields) < 4 {
			continue
		}

		n := Neighbor{}

		if ipStr := fields[1]; len(ipStr) < 2 {
			continue
		} else if ip := net.ParseIP(ipStr[1 : len(ipStr)-1]); ip == nil {
			continue
		} else {
			n.IP = ip
		}

		hwStr := fields[3]
		if mac, err := net.ParseMAC(hwStr); err != nil {
			continue
		} else {
			n.MAC = mac
		}

		host := fields[0]
		if err := netutil.ValidateDomainName(host); err != nil {
			log.Debug("parsing arp output: %s", err)
		} else {
			n.Name = host
		}

		ns = append(ns, n)
	}

	return ns
}
