//go:build windows

package aghnet

import (
	"io"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/sys/windows"
)

func canBindPrivilegedPorts() (can bool, err error) {
	return true, nil
}

func ifaceHasStaticIP(string) (ok bool, err error) {
	return false, aghos.Unsupported("checking static ip")
}

func ifaceSetStaticIP(string) (err error) {
	return aghos.Unsupported("setting static ip")
}

// closePortChecker closes c.  c must be non-nil.
func closePortChecker(c io.Closer) (err error) {
	if err = c.Close(); err != nil {
		return err
	}

	// It seems that net.Listener.Close() doesn't close file descriptors right
	// away.  We wait for some time and hope that this fd will be closed.
	//
	// TODO(e.burkov):  Investigate the purpose of the line and perhaps use more
	// reliable approach.
	time.Sleep(100 * time.Millisecond)

	return nil
}

func isAddrInUse(err syscall.Errno) (ok bool) {
	return errors.Is(err, windows.WSAEADDRINUSE)
}
