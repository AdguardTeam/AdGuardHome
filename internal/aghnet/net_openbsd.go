//go:build openbsd

package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
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
		if len(fields) >= 2 && fields[0] == "inet" && net.ParseIP(fields[1]) != nil {
			return nil, false, s.Err()
		}
	}

	return nil, true, s.Err()
}

func ifaceSetStaticIP(string) (err error) {
	return aghos.Unsupported("setting static ip")
}
