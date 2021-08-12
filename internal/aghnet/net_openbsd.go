//go:build openbsd
// +build openbsd

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

// maxCheckedFileSize is the maximum acceptable length of the /etc/hostname.*
// files.
const maxCheckedFileSize = 1024 * 1024

func ifaceHasStaticIP(ifaceName string) (ok bool, err error) {
	const filenameFmt = "/etc/hostname.%s"

	filename := fmt.Sprintf(filenameFmt, ifaceName)
	var f *os.File
	if f, err = os.Open(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}

		return false, err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	var r io.Reader
	r, err = aghio.LimitReader(f, maxCheckedFileSize)
	if err != nil {
		return false, err
	}

	return hostnameIfStaticConfig(r)
}

// hostnameIfStaticConfig checks if the interface is configured by
// /etc/hostname.* to have a static IP.
//
// TODO(e.burkov):  The platform-dependent functions to check the static IP
// address configured are rather similar.  Think about unifying common parts.
func hostnameIfStaticConfig(r io.Reader) (has bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "inet" && net.ParseIP(fields[1]) != nil {
			return true, s.Err()
		}
	}

	return false, s.Err()
}

func ifaceSetStaticIP(string) (err error) {
	return aghos.Unsupported("setting static ip")
}
