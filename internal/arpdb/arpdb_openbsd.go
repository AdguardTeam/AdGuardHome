//go:build openbsd

package arpdb

import (
	"bufio"
	"log/slog"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

func newARPDB(logger *slog.Logger) (arp *cmdARPDB) {
	return &cmdARPDB{
		logger: logger,
		parse:  parseArpA,
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
func parseArpA(logger *slog.Logger, sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
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

		n, err := newNeighbor("", fields[0], fields[1])
		if err != nil {
			logger.Debug("parsing arp output", "line", ln, slogutil.KeyError, err)

			continue
		}

		ns = append(ns, *n)
	}

	return ns
}
