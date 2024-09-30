//go:build openbsd

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
	filename := fmt.Sprintf("etc/hostname.%s", ifaceName)

	return aghos.FileWalker(hostnameIfStaticConfig).Walk(rootDirFS, filename)
}

// hostnameIfStaticConfig checks if the interface is configured by
// /etc/hostname.* to have a static IP.
func hostnameIfStaticConfig(r io.Reader) (_ []string, ok bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		switch {
		case
			len(fields) < 2,
			fields[0] != "inet",
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
