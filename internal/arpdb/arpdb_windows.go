//go:build windows

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
func parseArpA(logger *slog.Logger, sc *bufio.Scanner, lenHint int) (ns []Neighbor) {
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

		n, err := newNeighbor("", fields[0], fields[1])
		if err != nil {
			logger.Debug("parsing arp output", "line", ln, slogutil.KeyError, err)

			continue
		}

		ns = append(ns, *n)
	}

	return ns
}
