//go:build windows

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
		cmd:  "arp",
		args: []string{"/a"},
	}
}

// parseArpA parses the output of the "arp /a" command on Windows.  The expected
// input format (the first line is empty):
//
//	Interface: 192.168.56.16 --- 0x7
//	  Internet Address      Physical Address      Type
//	  192.168.56.1          0a-00-27-00-00-00     dynamic
//	  192.168.56.255        ff-ff-ff-ff-ff-ff     static
func parseArpA(sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
	ns = make([]Neighbor, 0, lenHint)
	for sc.Scan() {
		ln := sc.Text()
		if ln == "" {
			continue
		}

		fields := strings.Fields(ln)
		if len(fields) != 3 {
			continue
		}

		ip, err := netip.ParseAddr(fields[0])
		if err != nil {
			log.Debug("arpdb: parsing arp output: ip: %s", err)

			continue
		}

		mac, err := net.ParseMAC(fields[1])
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
