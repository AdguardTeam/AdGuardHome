//go:build darwin || freebsd || openbsd

package aghnet

import (
	"context"
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

	err = errors.Join(err, cerr)

	return errors.Annotate(err, "setting control options: %w")
}

// listenPacketReusable announces on the local network address additionally
// configuring the socket to have a reusable binding.
func listenPacketReusable(_, network, address string) (c net.PacketConn, err error) {
	var lc net.ListenConfig
	lc.Control = reuseAddrCtrl

	return lc.ListenPacket(context.Background(), network, address)
}
