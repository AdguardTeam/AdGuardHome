//go:build freebsd

package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/netutil"
)

func ifaceHasStaticIP(ifaceName string) (ok bool, err error) {
	const rcConfFilename = "etc/rc.conf"

	walker := aghos.FileWalker(interfaceName(ifaceName).rcConfStaticConfig)

	return walker.Walk(rootDirFS, rcConfFilename)
}

// rcConfStaticConfig checks if the interface is configured by /etc/rc.conf to
// have a static IP.
func (n interfaceName) rcConfStaticConfig(r io.Reader) (_ []string, cont bool, err error) {
	s := bufio.NewScanner(r)
	for pref := fmt.Sprintf("ifconfig_%s=", n); s.Scan(); {
		line := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(line, pref) {
			continue
		}

		cfgLeft, cfgRight := len(pref)+1, len(line)-1
		if cfgLeft >= cfgRight {
			continue
		}

		// TODO(e.burkov):  Expand the check to cover possible
		// configurations from man rc.conf(5).
		fields := strings.Fields(line[cfgLeft:cfgRight])
		switch {
		case
			len(fields) < 2,
			!strings.EqualFold(fields[0], "inet"),
			!netutil.IsValidIPString(fields[1]):
			continue
		default:
			return nil, false, s.Err()
		}
	}

	return nil, true, s.Err()
}

func ifaceSetStaticIP(string) (err error) {
	return aghos.Unsupported("setting static ip")
}
