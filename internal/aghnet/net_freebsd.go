//go:build freebsd
// +build freebsd

package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
)

func canBindPrivilegedPorts() (can bool, err error) {
	return aghos.HaveAdminRights()
}

// maxCheckedFileSize is the maximum acceptable length of the /etc/rc.conf file.
const maxCheckedFileSize = 1024 * 1024

func ifaceHasStaticIP(ifaceName string) (ok bool, err error) {
	const filename = "/etc/rc.conf"

	var f *os.File
	f, err = os.Open(filename)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	var r io.Reader
	r, err = aghio.LimitReader(f, maxCheckedFileSize)
	if err != nil {
		return false, err
	}

	return rcConfStaticConfig(r, ifaceName)
}

// rcConfStaticConfig checks if the interface is configured by /etc/rc.conf to
// have a static IP.
func rcConfStaticConfig(r io.Reader, ifaceName string) (has bool, err error) {
	s := bufio.NewScanner(r)
	for ifaceLinePref := fmt.Sprintf("ifconfig_%s", ifaceName); s.Scan(); {
		line := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(line, ifaceLinePref) {
			continue
		}

		eqIdx := len(ifaceLinePref)
		if line[eqIdx] != '=' {
			continue
		}

		fieldsStart, fieldsEnd := eqIdx+2, len(line)-1
		if fieldsStart >= fieldsEnd {
			continue
		}

		fields := strings.Fields(line[fieldsStart:fieldsEnd])
		if len(fields) >= 2 &&
			strings.ToLower(fields[0]) == "inet" &&
			net.ParseIP(fields[1]) != nil {
			return true, s.Err()
		}
	}

	return false, s.Err()
}

func ifaceSetStaticIP(string) (err error) {
	return aghos.Unsupported("setting static ip")
}
