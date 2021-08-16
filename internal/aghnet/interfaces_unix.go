//go:build aix || darwin || dragonfly || freebsd || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd netbsd openbsd solaris

package aghnet

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/sys/unix"
)

// reuseAddrCtrl is the function to be set to net.ListenConfig.Control.  It
// configures the socket to have a reusable port binding.
func reuseAddrCtrl(_, _ string, c syscall.RawConn) (err error) {
	cerr := c.Control(func(fd uintptr) {
		// TODO(e.burkov):  Consider using SO_REUSEPORT.
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			err = os.NewSyscallError("setsockopt", err)
		}
	})

	const (
		errMsg    = "setting control options"
		errMsgFmt = errMsg + ": %w"
	)

	if err != nil && cerr != nil {
		err = errors.List(errMsg, err, cerr)
	} else if err != nil {
		err = fmt.Errorf(errMsgFmt, err)
	} else if cerr != nil {
		err = fmt.Errorf(errMsgFmt, cerr)
	}

	return err
}

// listenPacketReusable announces on the local network address additionally
// configuring the socket to have a reusable binding.
func listenPacketReusable(_, network, address string) (c net.PacketConn, err error) {
	var lc net.ListenConfig
	lc.Control = reuseAddrCtrl

	return lc.ListenPacket(context.Background(), network, address)
}
